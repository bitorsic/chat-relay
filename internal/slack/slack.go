package slack

import (
	"bufio"
	"bytes"
	"context"
	"encoding/json"
	"log"
	"net/http"
	"os"
	"strings"

	"github.com/slack-go/slack"
	"github.com/slack-go/slack/slackevents"
	"github.com/slack-go/slack/socketmode"
	"go.opentelemetry.io/otel"
	"go.opentelemetry.io/otel/attribute"
	"go.opentelemetry.io/otel/codes"
	"go.opentelemetry.io/otel/propagation"
	"go.opentelemetry.io/otel/trace"
)

func StartSlackBot() {
	appToken := os.Getenv("SLACK_APP_TOKEN")
	if !strings.HasPrefix(appToken, "xapp-") {
		log.Fatalln("SLACK_APP_TOKEN must have the prefix \"xapp-\".")
	}

	botToken := os.Getenv("SLACK_BOT_TOKEN")
	if !strings.HasPrefix(botToken, "xoxb-") {
		log.Fatalln("SLACK_BOT_TOKEN must have the prefix \"xoxb-\".")
	}

	api := slack.New(
		botToken,
		slack.OptionDebug(false),
		slack.OptionAppLevelToken(appToken),
	)

	client := socketmode.New(
		api,
		socketmode.OptionDebug(false),
	)

	go func() {
		for evt := range client.Events {
			switch evt.Type {
			case socketmode.EventTypeConnecting:
				log.Println("Connecting to Slack with Socket Mode...")
			case socketmode.EventTypeConnectionError:
				log.Println("Connection failed. Retrying later...")
			case socketmode.EventTypeConnected:
				log.Println("Connected to Slack with Socket Mode")
			case socketmode.EventTypeEventsAPI:
				eventsAPIEvent, ok := evt.Data.(slackevents.EventsAPIEvent)
				if !ok {
					log.Printf("Ignored %+v\n", evt)

					continue
				}

				client.Ack(*evt.Request)

				switch eventsAPIEvent.Type {
				case slackevents.CallbackEvent:
					innerEvent := eventsAPIEvent.InnerEvent
					switch ev := innerEvent.Data.(type) {
					case *slackevents.AppMentionEvent:
						tracer := otel.Tracer("slack-bot")
						ctx, span := tracer.Start(context.Background(), "ReceivedSlackMessage")
						span.SetAttributes(
							attribute.String("event", "mention"),
						)
						span.End()

						log.Printf("Bot was mentioned in channel %v by user %v: %v", ev.Channel, ev.User, ev.Text)
						go handleSlackResponse(ctx, ev.Channel, ev.User, ev.Text, client)
					case *slackevents.MessageEvent:
						// ignore message that is not a DM (just for future safety), and message sent by bot itself
						if ev.ChannelType != "im" || ev.BotID != "" || ev.User == "" {
							continue
						}

						tracer := otel.Tracer("slack-bot")
						ctx, span := tracer.Start(context.Background(), "ReceivedSlackMessage")
						span.SetAttributes(
							attribute.String("event", "dm"),
						)
						span.End()

						log.Printf("Bot was DM'd (channel %v) by user %v: %v", ev.Channel, ev.User, ev.Text)
						go handleSlackResponse(ctx, ev.Channel, ev.User, ev.Text, client)
					}
				default:
					log.Println("unsupported Events API event received")
				}
			case socketmode.EventTypeHello:
				log.Println("Received hello from slack. Good to go now")
			default:
				log.Printf("Unexpected event type received: %s\n", evt.Type)
			}
		}
	}()

	client.Run()
}

func handleSlackResponse(ctx context.Context, channel string, user string, text string, client *socketmode.Client) {
	tracer := otel.Tracer("slack-bot")
	ctx, span := tracer.Start(ctx, "SendingPayload")

	httpMethod := "POST"
	endpoint := os.Getenv("BACKEND_URL") + "v1/chat/stream"

	payload := map[string]string{
		"user_id": user,
		"query":   text,
	}

	body, _ := json.Marshal(payload)

	span.SetAttributes(
		attribute.String("user_id", user),
		attribute.String("query", text),
		attribute.String("http_method", httpMethod),
		attribute.String("endpoint", endpoint),
	)

	// for propagating the trace to the backend as well
	propagator := propagation.TraceContext{}

	req, _ := http.NewRequestWithContext(ctx, httpMethod, endpoint, bytes.NewReader(body))
	req.Header.Set("Content-Type", "application/json")

	// injecting ctx into http header
	propagator.Inject(ctx, propagation.HeaderCarrier(req.Header))

	response, err := http.DefaultClient.Do(req)
	span.SetAttributes(attribute.Int("http_status_code", response.StatusCode))
	if err != nil {
		span.SetStatus(codes.Error, "Could not connect to the backend")
		span.RecordError(err)
		span.End()

		_, _, err := client.PostMessage(channel, slack.MsgOptionText("Could not connect to the backend. Please try again.", false))
		if err != nil {
			log.Printf("Failed to post message: %v", err)
		}
		return
	}
	defer response.Body.Close()

	span.End()

	ctx, span = tracer.Start(ctx, "ProcessBackendResponse")
	defer span.End()

	contentType := response.Header.Get("Content-Type")

	span.SetAttributes(attribute.String("content_type", contentType))

	// checking for prefix because header might include charset, etc.
	if strings.HasPrefix(contentType, "text/event-stream") {
		msgText := "..."

		_, timestamp, err := client.PostMessage(channel, slack.MsgOptionText(msgText, false))
		if err != nil {
			span.SetStatus(codes.Error, "Could not post message to slack")
			span.RecordError(err)

			log.Printf("Failed to post message: %v", err)
			return
		}

		scanner := bufio.NewScanner(response.Body)
		for scanner.Scan() {
			line := scanner.Text()

			if !strings.HasPrefix(line, "data: ") {
				continue
			}

			dataPart := strings.TrimPrefix(line, "data: ")

			var chunk map[string]string
			err := json.Unmarshal([]byte(dataPart), &chunk)
			if err != nil {
				log.Printf("could not parse json: %v", dataPart)
				return
			}

			textChunk, ok := chunk["text_chunk"]
			if !ok {
				status, ok := chunk["status"]
				if !ok {
					continue
				}

				span.AddEvent("ReceivedStatus", trace.WithAttributes(
					attribute.String("status", status),
				))

				if status == "done" {
					msgText = strings.TrimSuffix(msgText, "...")

					_, _, _, err = client.UpdateMessage(channel, timestamp, slack.MsgOptionText(msgText, false))
					if err != nil {
						span.SetStatus(codes.Error, "Could not update slack message")
						span.RecordError(err)

						log.Printf("Failed to update message: %v", err)
					}
					return
				}
			}

			span.AddEvent("ReceivedTextChunk")

			// formatting message text
			msgText = strings.TrimSuffix(msgText, "...")
			msgText += textChunk + "..."

			_, _, _, err = client.UpdateMessage(channel, timestamp, slack.MsgOptionText(msgText, false))
			if err != nil {
				span.SetStatus(codes.Error, "Could not update slack message")
				span.RecordError(err)

				log.Printf("Failed to update message: %v", err)
			}
		}
	} else if strings.HasPrefix(contentType, "application/json") {
		var data map[string]string

		err := json.NewDecoder(response.Body).Decode(&data)
		if err != nil {
			span.SetStatus(codes.Error, "Could not parse JSON response")
			span.RecordError(err)

			log.Println("received invalid json response")
			return
		}

		text := data["full_response"]

		_, _, err = client.PostMessage(channel, slack.MsgOptionText(text, false))
		if err != nil {
			span.SetStatus(codes.Error, "Could not post message to slack")
			span.RecordError(err)

			log.Printf("Failed to post message: %v", err)
		}

		span.AddEvent("ReceivedFullJSONResponse")
	}

}
