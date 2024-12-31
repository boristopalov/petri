package agent

import (
	"context"
	"fmt"
	"log"
	"os"
	"sync"
	"time"

	"github.com/boristopalov/petri/internal/client"
	"github.com/boristopalov/petri/pkg/core"
	"github.com/boristopalov/petri/pkg/messaging"
	"github.com/google/uuid"
)

// Agent represents an AI agent that can interact in experiments
type Agent interface {
	// Generate takes an observation and returns an action
	Generate(ctx context.Context) (string, error)
	// Observe allows the agent to process and store data
	Observe(ctx context.Context, data string) error
	// GetID returns the unique identifier for this agent
	GetID() string
	// GetModel returns information about the AI model being used
	GetModel() ModelInfo
}

type ModelInfo struct {
	Id     string         // e.g. "gpt-4o-mini"
	Config map[string]any // model-specific configuration
}

type LLMAgent struct {
	id            string
	model         ModelInfo
	client        LLMClient
	memory        *Memory
	config        map[string]any
	messageChan   chan messaging.Message
	messageBroker messaging.Broker
}

type LLMClient interface {
	Complete(ctx context.Context, model string, prompt string) (string, error)
}

type AgentParams struct {
	APIBaseUrl    string
	APIKey        string
	Model         ModelInfo
	AgentID       string
	MessageBroker messaging.Broker
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

	log.Println("PARAMS: %v", params)

	_client := client.GetOpenAiClient(params.APIBaseUrl, params.APIKey)

	agent := &LLMAgent{
		id:            params.AgentID,
		model:         params.Model,
		client:        _client,
		memory:        NewMemory(100), // short term memory - start with capacity of 100 events
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

// Send implements messaging.Sender
func (a *LLMAgent) Send(msg messaging.Message) error {
	msg.From = a.id
	msg.Timestamp = time.Now()
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
				log.Printf("Message from %s: %v", msg.From, msg.Content)
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

func (a *LLMAgent) GetClient() LLMClient {
	return a.client
}

func (a *LLMAgent) Generate(ctx context.Context, obs core.Observation) (core.Action, error) {
	// TODO: Implement action generation based on observation
	return core.Action{}, nil
}

func (a *LLMAgent) Observe(ctx context.Context, data string) error {
	return a.memory.Store(data)
}

type Memory struct {
	memoryStream []string
	capacity     int
	mu           sync.RWMutex
}

func NewMemory(capacity int) *Memory {
	return &Memory{
		memoryStream: make([]string, 0, capacity),
		capacity:     capacity,
	}
}

func (m *Memory) Store(data string) error {
	m.mu.Lock()
	defer m.mu.Unlock()

	m.memoryStream = append(m.memoryStream, data)

	// TODO: come up with a better solution for handling capacity limitations
	// It should likely be based on token counts
	if len(m.memoryStream) > m.capacity {
		m.memoryStream = m.memoryStream[1:]
	}
	return nil
}
