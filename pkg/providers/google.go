package providers

import (
	"context"
	"fmt"
	"os"

	"google.golang.org/genai"
)

type GeminiClient struct {
	client *genai.Client
}

func Gemini(ctx context.Context, params ProviderParams) (*GeminiClient, error) {
	apiKey := params.APIKey
	if apiKey == "" {
		apiKey = os.Getenv("GEMINI_API_KEY")
	}
	if apiKey == "" {
		return nil, fmt.Errorf("Error retrieving GEMINI_API_KEY")
	}
	client, err := genai.NewClient(ctx, &genai.ClientConfig{
		APIKey:  apiKey,
		Backend: genai.BackendGoogleAI,
	})
	if err != nil {
		return nil, err
	}
	return &GeminiClient{
		client: client,
	}, nil
}

func (c *GeminiClient) Complete(ctx context.Context, model string, prompt string) (string, error) {
	parts := []*genai.Part{
		{Text: prompt},
	}
	result, err := c.client.Models.GenerateContent(ctx, "gemini-2.0-flash-exp", []*genai.Content{{Parts: parts}}, nil)
	if err != nil {
		return "", err
	}
	return result.PromptFeedback.BlockReasonMessage, nil
}
