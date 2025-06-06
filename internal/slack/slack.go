package slack

import (
	"bufio"
	"bytes"
	"encoding/json"
	"log"
	"net/http"
	"os"
	"strings"

	"github.com/slack-go/slack"
	"github.com/slack-go/slack/slackevents"
	"github.com/slack-go/slack/socketmode"
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
				log.Println("Connected to Slack with Socket Mode.")
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
						log.Printf("Bot was mentioned in channel %v by user %v: %v", ev.Channel, ev.User, ev.Text)
						go handleSlackResponse(ev.Channel, ev.User, ev.Text, client)
					case *slackevents.MessageEvent:
						// ignore message that is not a DM (just for future safety), and message sent by bot itself
						if ev.ChannelType != "im" || ev.BotID != "" || ev.User == "" {
							continue
						}

						log.Printf("Bot was DM'd (channel %v) by user %v: %v", ev.Channel, ev.User, ev.Text)
						go handleSlackResponse(ev.Channel, ev.User, ev.Text, client)
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

func handleSlackResponse(channel string, user string, text string, client *socketmode.Client) {
	payload := map[string]string{
		"user_id": user,
		"query":   text,
	}

	body, _ := json.Marshal(payload)
	req, _ := http.NewRequest("POST", os.Getenv("BACKEND_URL")+"v1/chat/stream", bytes.NewReader(body))
	req.Header.Set("Content-Type", "application/json")

	response, err := http.DefaultClient.Do(req)
	if err != nil {
		_, _, err := client.PostMessage(channel, slack.MsgOptionText("Could not connect to the backend. Please try again.", false))
		if err != nil {
			log.Printf("Failed to post message: %v", err)
		}
	}
	defer response.Body.Close()

	contentType := response.Header.Get("Content-Type")

	// checking for prefix because header might include charset, etc.
	if strings.HasPrefix(contentType, "text/event-stream") {
		msgText := "..."

		_, timestamp, err := client.PostMessage(channel, slack.MsgOptionText(msgText, false))
		if err != nil {
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

				if status == "done" {
					msgText = strings.TrimSuffix(msgText, "...")

					_, _, _, err = client.UpdateMessage(channel, timestamp, slack.MsgOptionText(msgText, false))
					if err != nil {
						log.Printf("Failed to update message: %v", err)
					}
					return
				}
			}

			// formatting message text
			msgText = strings.TrimSuffix(msgText, "...")
			msgText += textChunk + "..."

			_, _, _, err = client.UpdateMessage(channel, timestamp, slack.MsgOptionText(msgText, false))
			if err != nil {
				log.Printf("Failed to update message: %v", err)
			}
		}
	} else if strings.HasPrefix(contentType, "application/json") {
		var data map[string]string

		err := json.NewDecoder(response.Body).Decode(&data)
		if err != nil {
			log.Println("received invalid json response")
			return
		}

		text := data["full_response"]

		_, _, err = client.PostMessage(channel, slack.MsgOptionText(text, false))
		if err != nil {
			log.Printf("Failed to post message: %v", err)
		}
	}

}
