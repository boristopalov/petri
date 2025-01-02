package providers

import (
	"context"
	"fmt"
	"os"

	"github.com/openai/openai-go"
	"github.com/openai/openai-go/option"
)

type openAIClient struct {
	client *openai.Client
}

func OpenAi(ctx context.Context, opts ...ProviderOption) (*openAIClient, error) {
	params := &ProviderParams{}

	// Apply all options
	for _, opt := range opts {
		opt(params)
	}

	// Set defaults and environment fallbacks
	baseUrl := params.BaseURL
	if baseUrl == "" {
		baseUrl = os.Getenv("OPENAI_API_BASE_URL")
		if baseUrl == "" {
			baseUrl = "https://api.openai.com/v1/" // Default OpenAI API endpoint
		}
	}
	apiKey := params.APIKey
	if apiKey == "" {
		apiKey = os.Getenv("OPENAI_API_KEY")
	}
	if apiKey == "" {
		return nil, fmt.Errorf("Error retrieving OPENAI_API_KEY")
	}
	client := openai.NewClient(
		option.WithAPIKey(apiKey),
		option.WithBaseURL(baseUrl),
	)
	return &openAIClient{
		client: client,
	}, nil
}

func (c *openAIClient) Complete(ctx context.Context, model string, prompt string) (string, error) {
	chatCompletion, err := c.client.Chat.Completions.New(ctx, openai.ChatCompletionNewParams{
		Messages: openai.F([]openai.ChatCompletionMessageParamUnion{
			openai.UserMessage(prompt),
		}),
		Model: openai.F(model),
	})
	if err != nil {
		return "", err
	}
	return chatCompletion.Choices[0].Message.Content, nil
}
