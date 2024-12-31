package agent

import (
	"context"
	"fmt"
	"log"
	"os"
	"strings"
	"time"

	"github.com/boristopalov/petri/internal/client"
	"github.com/boristopalov/petri/pkg/memory"
	"github.com/boristopalov/petri/pkg/messaging"
	"github.com/google/uuid"
)

// Agent represents an AI agent that can interact in experiments
type Agent interface {
	// Runs the agent with a prompt
	Run(ctx context.Context) (string, error)
}

type ModelInfo struct {
	Id     string         // e.g. "gpt-4o-mini"
	Config map[string]any // model-specific configuration
}

type LLMAgent struct {
	id            string
	model         ModelInfo
	task          string
	client        ApiClient
	memory        *memory.Memory
	config        map[string]any
	messageChan   chan messaging.Message
	messageBroker messaging.Broker
}

type ApiClient interface {
	Complete(ctx context.Context, model string, prompt string) (string, error)
}

type AgentParams struct {
	APIBaseUrl    string
	APIKey        string
	Model         ModelInfo
	AgentID       string
	MessageBroker messaging.Broker
	Task          string
}

type AgentOption func(*AgentParams)

func WithAPIBaseURL(url string) AgentOption {
	return func(p *AgentParams) {
		p.APIBaseUrl = url
	}
}

func WithAPIKey(key string) AgentOption {
	return func(p *AgentParams) {
		p.APIKey = key
	}
}

func WithModel(model ModelInfo) AgentOption {
	return func(p *AgentParams) {
		p.Model = model
	}
}

func WithAgentId(id string) AgentOption {
	return func(p *AgentParams) {
		p.AgentID = id
	}
}

func WithMessageBroker(b messaging.Broker) AgentOption {
	return func(p *AgentParams) {
		p.MessageBroker = b
	}
}

func WithTask(task string) AgentOption {
	return func(p *AgentParams) {
		p.Task = task
	}
}

func defaultAgentParams() *AgentParams {
	return &AgentParams{
		APIBaseUrl: "https://api.openai.com/v1/",
		APIKey:     os.Getenv("OPENAI_API_KEY"),
		Model: ModelInfo{
			Id:     "gpt-4o-mini",
			Config: make(map[string]any),
		},
		AgentID: "agent-" + uuid.New().String(),
	}
}

// NewLLMAgent creates a new LLM agent
func NewLLMAgent(opts ...AgentOption) (*LLMAgent, error) {
	params := defaultAgentParams()

	for _, opt := range opts {
		opt(params)
	}

	_client := client.GetOpenAiClient(params.APIBaseUrl, params.APIKey)

	agent := &LLMAgent{
		id:            params.AgentID,
		task:          params.Task,
		model:         params.Model,
		client:        _client,
		memory:        memory.NewMemory(100), // short term memory - start with capacity of 100 events
		config:        make(map[string]any),
		messageChan:   make(chan messaging.Message, 100), // Buffer 100 messages
		messageBroker: params.MessageBroker,
	}

	// Subscribe to messages
	if err := agent.messageBroker.Subscribe(agent.id, agent.messageChan); err != nil {
		// Handle error appropriately
		panic(err)
	}

	return agent, nil
}

func (a *LLMAgent) GetID() string {
	return a.id
}

func (a *LLMAgent) GetModel() ModelInfo {
	return a.model
}

func (a *LLMAgent) GetClient() ApiClient {
	return a.client
}

// Send implements messaging.Sender
func (a *LLMAgent) Send(msg messaging.Message) error {
	msg.From = a.id
	msg.Timestamp = time.Now()
	log.Printf("[%s]: %s\n\n", a.id, msg.Content)
	return a.messageBroker.Publish(msg)
}

// Receive implements messaging.Receiver
func (a *LLMAgent) Receive() <-chan messaging.Message {
	return a.messageChan
}

// StartMessageHandler starts a goroutine to handle incoming messages
func (a *LLMAgent) StartMessageHandler(ctx context.Context) {
	go func() {
		for {
			select {
			case msg := <-a.messageChan:
				// Store the message in memory
				if err := a.memory.Store(fmt.Sprintf("Message from %s: %v", msg.From, msg.Content)); err != nil {
					log.Printf("Failed to store message in memory: %v", err)
				}
			case <-ctx.Done():
				return
			}
		}
	}()
}

func (a *LLMAgent) Run(ctx context.Context) (string, error) {
	// Generate a response based on memory and task
	memories := a.memory.GetAllMessages()
	var prompt string
	if len(memories) == 0 {
		prompt = fmt.Sprintf("You are %s. Your task is: %s\n\n Begin!",
			a.id,
			a.task)

	} else {
		prompt = fmt.Sprintf("You are %s. Your task is: %s\n\nRecent conversation history:\n%s\n\nBased on this context, generate a response:",
			a.id,
			a.task,
			strings.Join(memories, "\n"))
	}

	response, err := a.client.Complete(ctx, a.model.Id, prompt)
	if err != nil {
		return "", fmt.Errorf("failed to generate response: %v", err)
	}

	// Send the response through the message broker
	err = a.Send(messaging.Message{
		Content: response,
		To:      []string{}, // broadcast to all
	})
	if err != nil {
		return "", fmt.Errorf("failed to send message: %v", err)
	}

	return response, nil
}
