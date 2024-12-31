package messaging

import (
	"fmt"
	"sync"
)

// SimpleBroker implements the Broker interface
// subscribers is a map where keys are agent IDs and values are channels for receiving messages
type SimpleBroker struct {
	subscribers map[string]chan<- Message
	mu          sync.RWMutex
}

// NewBroker creates a new message broker
func NewBroker() *SimpleBroker {
	return &SimpleBroker{
		subscribers: make(map[string]chan<- Message),
	}
}

// Publish sends a message to specified recipients
func (b *SimpleBroker) Publish(msg Message) error {
	b.mu.RLock()
	defer b.mu.RUnlock()

	// If no recipients specified, broadcast to all subscribers
	recipients := msg.To
	if len(recipients) == 0 {
		for id := range b.subscribers {
			if id != msg.From { // Don't send to self
				recipients = append(recipients, id)
			}
		}
	}

	// Send to each recipient
	for _, recipientID := range recipients {
		ch, ok := b.subscribers[recipientID]
		if !ok {
			continue // Skip if recipient not found
		}

		// Non-blocking send
		select {
		case ch <- msg:
			// Message sent successfully
		default:
			// Channel is full, skip this message
			return fmt.Errorf("recipient %s's channel is full", recipientID)
		}
	}

	return nil
}

// Subscribe registers an agent to receive messages
func (b *SimpleBroker) Subscribe(agentID string, ch chan<- Message) error {
	b.mu.Lock()
	defer b.mu.Unlock()

	if _, exists := b.subscribers[agentID]; exists {
		return fmt.Errorf("agent %s is already subscribed", agentID)
	}

	b.subscribers[agentID] = ch
	return nil
}

// Unsubscribe removes an agent's subscription
func (b *SimpleBroker) Unsubscribe(agentID string) error {
	b.mu.Lock()
	defer b.mu.Unlock()

	if _, exists := b.subscribers[agentID]; !exists {
		return fmt.Errorf("agent %s is not subscribed", agentID)
	}

	delete(b.subscribers, agentID)
	return nil
}

func (b *SimpleBroker) Reset() {
	b.mu.Lock()
	defer b.mu.Unlock()
	b.subscribers = make(map[string]chan<- Message)
}
