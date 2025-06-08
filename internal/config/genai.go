package config

import (
	"context"
	"os"

	"google.golang.org/genai"
)

var GenAIClient *genai.Client
var GenAIConfig *genai.GenerateContentConfig

func GenAISetup(ctx context.Context) {
	GenAIClient, _ = genai.NewClient(ctx, &genai.ClientConfig{
		APIKey:  os.Getenv("GEMINI_API_KEY"),
		Backend: genai.BackendGeminiAPI,
	})

	GenAIConfig = &genai.GenerateContentConfig{
		SystemInstruction: genai.NewContentFromText("You are a slack bot named \"ChatRelay\". You will either get DMs or get mentioned in a channel.", genai.RoleUser),
	}
}
