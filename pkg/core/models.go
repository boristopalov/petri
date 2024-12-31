package core

import (
	"time"
)

type Action struct {
	AgentID   string
	Type      string
	Content   any
	Timestamp time.Time
}

type Observation struct {
	Type      string
	Content   any
	Timestamp time.Time
	SourceID  string
}

// type Event struct {
// 	Type      string
// 	Content   any
// 	Timestamp time.Time
// 	AgentID   string
// }

type State struct {
	Agents      map[string]AgentState
	Environment map[string]any
	Timestamp   time.Time
}

type AgentState struct {
	// Agent-specific state information
	Resources map[string]any
	Status    string
}

type ExperimentStatus struct {
	Running   bool
	StartTime time.Time
	EndTime   time.Time
	Errors    []error
}

// Message represents a communication between agents
type Message struct {
	From      string    // Agent ID of sender
	To        []string  // Agent IDs of recipients (empty means broadcast)
	Content   any       // The actual message content
	Timestamp time.Time // When the message was sent
}

// MessageBroker handles message routing between agents
type MessageBroker interface {
	// Publish sends a message to specified recipients
	Publish(msg Message) error
	// Subscribe registers an agent to receive messages
	Subscribe(agentID string, ch chan<- Message) error
	// Unsubscribe removes an agent's subscription
	Unsubscribe(agentID string) error
}
