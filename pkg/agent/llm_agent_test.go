package agent

import (
	"context"
	"fmt"
	"os"
	"path/filepath"
	"testing"
	"time"

	"github.com/boristopalov/petri/pkg/messaging"
	"github.com/joho/godotenv"
)

// MockLLMClient implements LLMClient interface for testing
type MockLLMClient struct{}

func (m *MockLLMClient) Complete(ctx context.Context, model string, prompt string) (string, error) {
	return "mock response", nil
}

func init() {
	envFilePath := filepath.Join("../../.env")

	if err := godotenv.Load(envFilePath); err != nil {
		panic(err)
	}
}

func TestLLMAgent(t *testing.T) {
	// Skip if no API key is set
	apiKey := os.Getenv("OPENAI_API_KEY")
	if apiKey == "" {
		t.Fatalf("OPENAI_API_KEY not set")
	}

	// Create a new agent
	agent, err := NewLLMAgent(
		WithAgentId("test-agent"),
		WithModel(ModelInfo{Id: "gpt-4o-mini", Config: make(map[string]any)}),
	)

	agent.client = &MockLLMClient{}

	if err != nil {
		t.Fatalf("Failed to create agent: %v", err)
	}
	if agent == nil {
		t.Fatal("Agent is nil")
	}

	// Test basic agent properties
	if got := agent.GetID(); got != "test-agent" {
		t.Errorf("agent.GetID() = %v, want %v", got, "test-agent")
	}
	if got := agent.GetModel().Id; got != "gpt-4o-mini" {
		t.Errorf("agent.GetModel().Id = %v, want %v", got, "gpt-4o-mini")
	}

	// Test API connection by making a simple completion request
	ctx := context.Background()
	response, err := agent.client.Complete(ctx, agent.model.Id, "Say hello!")

	if err != nil {
		t.Fatalf("Failed to complete request: %v", err)
	}
	if response != "mock response" {
		t.Error("Incorrect Response")
	}
}

func TestAgentMessaging(t *testing.T) {
	// Create two agents with mock clients
	agent1, err := NewLLMAgent(WithAgentId("agent1"), WithModel(ModelInfo{
		Id:     "mock-model",
		Config: make(map[string]any),
	}))
	if err != nil {
		t.Fatalf("Failed to create agent1: %v", err)
	}
	agent1.client = &MockLLMClient{} // Replace with mock client

	agent2, err := NewLLMAgent(WithAgentId("agent2"), WithModel(ModelInfo{
		Id:     "mock-model",
		Config: make(map[string]any),
	}),
	)
	if err != nil {
		t.Fatalf("Failed to create agent2: %v", err)
	}
	agent2.client = &MockLLMClient{} // Replace with mock client

	// Test direct message
	t.Run("direct message between agents", func(t *testing.T) {
		message := messaging.Message{
			Content: "Hello agent2!",
			To:      []string{"agent2"},
		}

		// Send message from agent1 to agent2
		if err := agent1.Send(message); err != nil {
			t.Fatalf("Failed to send message: %v", err)
		}

		// Wait for agent2 to receive the message
		select {
		case received := <-agent2.Receive():
			if received.From != "agent1" {
				t.Errorf("Expected message from agent1, got %s", received.From)
			}
			if received.Content != "Hello agent2!" {
				t.Errorf("Expected content 'Hello agent2!', got %v", received.Content)
			}
		case <-time.After(time.Second):
			t.Error("Timeout waiting for message")
		}

		// Agent1 should not receive their own message
		select {
		case msg := <-agent1.Receive():
			t.Errorf("Agent1 should not receive their own message: %v", msg)
		case <-time.After(100 * time.Millisecond):
			// This is expected
		}
	})

	// Test broadcast message
	t.Run("broadcast message between agents", func(t *testing.T) {
		message := messaging.Message{
			Content: "Hello everyone!",
			To:      []string{}, // empty To field means broadcast
		}

		// Send broadcast from agent1
		if err := agent1.Send(message); err != nil {
			t.Fatalf("Failed to send broadcast message: %v", err)
		}

		// Agent2 should receive the broadcast
		select {
		case received := <-agent2.Receive():
			if received.From != "agent1" {
				t.Errorf("Expected broadcast from agent1, got %s", received.From)
			}
			if received.Content != "Hello everyone!" {
				t.Errorf("Expected content 'Hello everyone!', got %v", received.Content)
			}
		case <-time.After(time.Second):
			t.Error("Timeout waiting for broadcast message")
		}

		// Agent1 should not receive their own broadcast
		select {
		case msg := <-agent1.Receive():
			t.Errorf("Agent1 should not receive their own broadcast: %v", msg)
		case <-time.After(100 * time.Millisecond):
			// This is expected
		}
	})

	// Test message storage in memory
	t.Run("test message storage in memory", func(t *testing.T) {
		// Start message handler
		ctx, cancel := context.WithCancel(context.Background())
		defer cancel()
		agent1.StartMessageHandler(ctx)

		// Send a message to the agent
		msg := messaging.Message{
			From:      "agent2",
			Content:   "Test message",
			To:        []string{"agent1"},
			Timestamp: time.Now(),
		}

		// Publish directly through broker to simulate receiving a message
		if err := agent1.messageBroker.Publish(msg); err != nil {
			t.Fatalf("Failed to publish message: %v", err)
		}

		// Wait a bit for message to be processed
		time.Sleep(100 * time.Millisecond)

		// Verify message is in memory
		found := false
		expectedContent := fmt.Sprintf("Message from %s: %v", msg.From, msg.Content)
		for _, stored := range agent1.memory.GetAllMessages() {
			if stored == expectedContent {
				found = true
				break
			}
		}

		if !found {
			t.Errorf("Message not found in memory. Expected: %s", expectedContent)
			t.Errorf("Memory contents: %v", agent1.memory.GetAllMessages())
		}
	})
}
