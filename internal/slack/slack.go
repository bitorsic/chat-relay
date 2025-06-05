package slack

import (
	"fmt"
	"log"
	"strings"

	"github.com/slack-go/slack"
	"github.com/slack-go/slack/slackevents"
	"github.com/slack-go/slack/socketmode"
)

func StartSlackBot(appToken string, botToken string) {
	if !strings.HasPrefix(appToken, "xapp-") {
		log.Fatalln("SLACK_APP_TOKEN must have the prefix \"xapp-\".")
	}

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
						if ev.ChannelType != "im" || ev.BotID != "" {
							continue
						}

						log.Printf("Bot was DM'd (channel %v) by user %v: %v", ev.Channel, ev.User, ev.Text)
						go handleSlackResponse(ev.Channel, ev.User, ev.Text, client)
					}
				default:
					client.Debugf("unsupported Events API event received")
				}
			case socketmode.EventTypeHello:
				client.Debugf("Hello received!")
			default:
				log.Printf("Unexpected event type received: %s\n", evt.Type)
			}
		}
	}()

	client.Run()
}

func handleSlackResponse(channel string, user string, text string, client *socketmode.Client) {
	fmt.Printf("Channel: %v\nUser: %v\nText: %v\n", channel, user, text)

	_, _, err := client.PostMessage(channel, slack.MsgOptionText("Well hello there fellow user", false))
	if err != nil {
		log.Printf("Failed to post message: %v", err)
	}
}
