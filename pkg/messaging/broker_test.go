package messaging

import (
	"testing"
	"time"
)

func TestBroker(t *testing.T) {
	t.Run("test direct message", func(t *testing.T) {
		broker := NewBroker()
		t.Cleanup(func() {
			broker.Reset()
		})
		ch1 := make(chan Message, 1)
		ch2 := make(chan Message, 1)

		if err := broker.Subscribe("agent1", ch1); err != nil {
			t.Fatalf("Failed to subscribe agent1: %v", err)
		}
		if err := broker.Subscribe("agent2", ch2); err != nil {
			t.Fatalf("Failed to subscribe agent2: %v", err)
		}

		msg := Message{
			From:      "agent1",
			To:        []string{"agent2"},
			Content:   "Hello agent2",
			Timestamp: time.Now(),
		}

		if err := broker.Publish(msg); err != nil {
			t.Fatalf("Failed to publish message: %v", err)
		}

		// agent2 should receive the message
		select {
		case received := <-ch2:
			if received.From != "agent1" || received.Content != "Hello agent2" {
				t.Errorf("Unexpected message received: %+v", received)
			}
		case <-time.After(time.Second):
			t.Error("Timeout waiting for message")
		}

		// agent1 should not receive the message
		select {
		case msg := <-ch1:
			t.Errorf("agent1 should not receive message but got: %+v", msg)
		case <-time.After(100 * time.Millisecond):
			// This is expected
		}
	})

	t.Run("test broadcast message", func(t *testing.T) {
		broker := NewBroker()
		t.Cleanup(func() {
			broker.Reset()
		})
		ch1 := make(chan Message, 1)
		ch2 := make(chan Message, 1)
		ch3 := make(chan Message, 1)

		agents := map[string]chan Message{
			"agent1": ch1,
			"agent2": ch2,
			"agent3": ch3,
		}

		for id, ch := range agents {
			if err := broker.Subscribe(id, ch); err != nil {
				t.Fatalf("Failed to subscribe %s: %v", id, err)
			}
		}

		msg := Message{
			From:      "agent1",
			To:        []string{}, // broadcast
			Content:   "Hello everyone",
			Timestamp: time.Now(),
		}

		if err := broker.Publish(msg); err != nil {
			t.Fatalf("Failed to publish broadcast message: %v", err)
		}

		// agent2 and agent3 should receive the message, but not agent1 (sender)
		for id, ch := range agents {
			if id == "agent1" {
				// Sender should not receive their own broadcast
				select {
				case msg := <-ch:
					t.Errorf("Sender received their own broadcast: %+v", msg)
				case <-time.After(100 * time.Millisecond):
					// This is expected
				}
			} else {
				select {
				case received := <-ch:
					if received.From != "agent1" || received.Content != "Hello everyone" {
						t.Errorf("Unexpected message received by %s: %+v", id, received)
					}
				case <-time.After(time.Second):
					t.Errorf("Timeout waiting for broadcast message on %s", id)
				}
			}
		}
	})

	t.Run("test subscription management", func(t *testing.T) {
		broker := NewBroker()
		t.Cleanup(func() {
			broker.Reset()
		})
		ch := make(chan Message, 1)

		// Test subscribe
		if err := broker.Subscribe("agent1", ch); err != nil {
			t.Fatalf("Failed to subscribe: %v", err)
		}

		// Test duplicate subscription
		if err := broker.Subscribe("agent1", ch); err == nil {
			t.Error("Expected error for duplicate subscription, got nil")
		}

		// Test unsubscribe
		if err := broker.Unsubscribe("agent1"); err != nil {
			t.Fatalf("Failed to unsubscribe: %v", err)
		}

		// Test unsubscribe non-existent agent
		if err := broker.Unsubscribe("agent1"); err == nil {
			t.Error("Expected error for unsubscribing non-existent agent, got nil")
		}
	})

	t.Run("test channel full behavior", func(t *testing.T) {
		broker := NewBroker()
		t.Cleanup(func() {
			broker.Reset()
		})
		ch := make(chan Message, 1) // Buffer size of 1

		if err := broker.Subscribe("agent1", ch); err != nil {
			t.Fatalf("Failed to subscribe: %v", err)
		}

		msg := Message{
			From:      "agent2",
			To:        []string{"agent1"},
			Content:   "Message 1",
			Timestamp: time.Now(),
		}

		// Fill the channel
		if err := broker.Publish(msg); err != nil {
			t.Fatalf("Failed to publish first message: %v", err)
		}

		// Try to send another message to full channel
		msg.Content = "Message 2"
		if err := broker.Publish(msg); err == nil {
			t.Error("Expected error when publishing to full channel, got nil")
		}
	})
}
