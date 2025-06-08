package backend

import (
	"chat-relay/internal/config"
	"context"
	"iter"

	"google.golang.org/genai"
)

func getGenAIStream(ctx context.Context, prompt string) iter.Seq2[*genai.GenerateContentResponse, error] {
	stream := config.GenAIClient.Models.GenerateContentStream(
		ctx,
		"gemini-1.5-flash",
		genai.Text(prompt),
		config.GenAIConfig,
	)

	return stream
}

func getGenAIFullResponse(ctx context.Context, prompt string) (string, error) {
	result, err := config.GenAIClient.Models.GenerateContent(
		ctx,
		"gemini-1.5-flash",
		genai.Text(prompt),
		config.GenAIConfig,
	)
	if err != nil {
		return "", err
	}

	return result.Text(), nil
}
