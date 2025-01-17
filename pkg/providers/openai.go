package providers

import (
	"context"
	"fmt"
	"log"
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
		return nil, fmt.Errorf("error retrieving OPENAI_API_KEY")
	}
	client := openai.NewClient(
		option.WithAPIKey(apiKey),
		option.WithBaseURL(baseUrl),
	)
	return &openAIClient{
		client: client,
	}, nil
}

func (c *openAIClient) Complete(ctx context.Context, model string, prompt string, systemPrompt string, history []string) (string, error) {
	log.Printf("Making OpenAI API call with model: %s", model)

	messages := []openai.ChatCompletionMessageParamUnion{
		openai.SystemMessage(systemPrompt),
	}

	// Add history as assistant messages
	for _, msg := range history {
		messages = append(messages, openai.AssistantMessage(msg))
	}

	// Add current prompt as the final user message
	messages = append(messages, openai.UserMessage(prompt))

	chatCompletion, err := c.client.Chat.Completions.New(ctx, openai.ChatCompletionNewParams{
		Messages: openai.F(messages),
		Model:    openai.F(model),
	})
	if err != nil {
		log.Printf("OpenAI API error: %v", err)
		return "", err
	}
	return chatCompletion.Choices[0].Message.Content, nil
}
