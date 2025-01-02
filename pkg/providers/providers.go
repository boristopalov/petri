package providers

import (
	"context"
	"fmt"
	"log"
	"os"
	"sync"

	"github.com/openai/openai-go"
	"github.com/openai/openai-go/option"

	"google.golang.org/genai"
)

type OpenAIClient struct {
	client *openai.Client
}

type GeminiClient struct {
	client *genai.Client
}

var (
	once   sync.Once
	client *OpenAIClient
)

type ProviderParams struct {
	BaseURL string
	APIKey  string
}

type OpenAIOption func(*ProviderParams)

func WithBaseURL(baseURL string) OpenAIOption {
	return func(p *ProviderParams) {
		p.BaseURL = baseURL
	}
}

func WithAPIKey(apiKey string) OpenAIOption {
	return func(p *ProviderParams) {
		p.APIKey = apiKey
	}
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

func OpenAi(ctx context.Context, opts ...OpenAIOption) *OpenAIClient {
	// Use singleton pattern to ensure only one client instance
	once.Do(func() {
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
		client = newOpenAIClient(ctx, *params)
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
