package providers

import (
	"context"
	"log"
	"os"

	"github.com/openai/openai-go"
	"github.com/openai/openai-go/option"
)

type OpenAIClient struct {
	client *openai.Client
}

func newOpenAIClient(ctx context.Context, params ProviderParams) *OpenAIClient {
	var client *openai.Client
	if params.BaseURL == "" {
		params.BaseURL = "https://api.openai.com/v1/"
	}
	if params.APIKey != "" {
		client = openai.NewClient(
			option.WithAPIKey(params.APIKey),
			option.WithBaseURL(params.BaseURL),
		)
	} else {
		client = openai.NewClient(
			option.WithBaseURL(params.BaseURL),
		)
	}
	log.Println("Using Base URL", params.BaseURL)
	return &OpenAIClient{
		client: client,
	}
}

func OpenAi(ctx context.Context, opts ...ProviderOption) *OpenAIClient {
	// Use singleton pattern to ensure only one client instance
	params := &ProviderParams{}

	// Apply all options
	for _, opt := range opts {
		opt(params)
	}

	// Set defaults and environment fallbacks
	if params.BaseURL == "" {
		params.BaseURL = os.Getenv("OPENAI_API_BASE_URL")
		if params.BaseURL == "" {
			params.BaseURL = "https://api.openai.com/v1/" // Default OpenAI API endpoint
		}
	}
	if params.APIKey == "" {
		params.APIKey = os.Getenv("OPENAI_API_KEY")
	}
	return newOpenAIClient(ctx, *params)
}

func (c *OpenAIClient) Complete(ctx context.Context, model string, prompt string) (string, error) {
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
