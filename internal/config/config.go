package config

import (
	"fmt"
	"os"
)

type Config struct {
	BackendPort      string
	SlackAppToken    string
	SlackBotToken    string
	StreamedResponse bool
}

func Load() (*Config, error) {
	backendPort := os.Getenv("BACKEND_PORT")
	slackAppToken := os.Getenv("SLACK_APP_TOKEN")
	slackBotToken := os.Getenv("SLACK_BOT_TOKEN")

	if backendPort == "" || slackAppToken == "" || slackBotToken == "" {
		return nil, fmt.Errorf(".env not complete")
	}

	return &Config{
		BackendPort:      backendPort,
		SlackAppToken:    slackAppToken,
		SlackBotToken:    slackBotToken,
		StreamedResponse: true,
	}, nil
}
