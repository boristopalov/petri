package memory

import "sync"

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

// GetAllMessages returns a copy of all messages in memory
func (m *Memory) GetAllMessages() []string {
	m.mu.RLock()
	defer m.mu.RUnlock()

	// Return a copy to prevent external modifications
	messages := make([]string, len(m.memoryStream))
	copy(messages, m.memoryStream)
	return messages
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
