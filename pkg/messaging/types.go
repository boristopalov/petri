package messaging

import (
	"time"
)

// Message represents a communication between agents
type Message struct {
	From      string    // Agent ID of sender
	To        []string  // Agent IDs of recipients (empty means broadcast)
	Content   any       // The actual message content
	Timestamp time.Time // When the message was sent
}

// Sender can send messages
type Sender interface {
	Send(msg Message) error
}

// Receiver can receive messages
type Receiver interface {
	Receive() <-chan Message
}

// Agent combines sending and receiving capabilities
type Agent interface {
	Sender
	Receiver
}

// Broker handles message routing between agents
type Broker interface {
	// Publish sends a message to specified recipients
	Publish(msg Message) error
	// Subscribe registers an agent to receive messages
	Subscribe(agentID string, ch chan<- Message) error
	// Unsubscribe removes an agent's subscription
	Unsubscribe(agentID string) error
}
