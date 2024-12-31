package client

import (
	"context"
	"log"
	"os"
	"sync"

	"github.com/openai/openai-go"
	"github.com/openai/openai-go/option"
)

type OpenAIClient struct {
	client *openai.Client
}

var (
	once   sync.Once
	client *OpenAIClient
)

func newOpenAIClient(baseUrl string, apiKey string) *OpenAIClient {
	var client *openai.Client
	if apiKey != "" {
		client = openai.NewClient(
			option.WithAPIKey(apiKey),
			option.WithBaseURL(baseUrl),
		)
	} else {
		client = openai.NewClient(
			option.WithBaseURL(baseUrl),
		)
	}
	log.Println("Using Base URL", baseUrl)
	return &OpenAIClient{
		client: client,
	}
}

func GetOpenAiClient(baseUrl string, apiKey string) *OpenAIClient {
	// Use singleton pattern to ensure only one client instance
	once.Do(func() {
		if baseUrl == "" {
			baseUrl = os.Getenv("OPENAI_API_BASE_URL")
			if baseUrl == "" {
				baseUrl = "https://api.openai.com/v1/" // Default OpenAI API endpoint
			}
		}
		client = newOpenAIClient(baseUrl, apiKey)
	})
	return client
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
