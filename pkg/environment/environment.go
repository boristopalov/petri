package environment

import (
	"context"
	"fmt"
	"log"
	"sync"
	"time"

	"github.com/boristopalov/petri/pkg/agent"
)

// State represents the basic state any environment must track
type State interface {
	GetStatus() string
	GetStep() uint32
	GetTimestamp() time.Time
}

// BaseState provides a basic implementation of State
type BaseState struct {
	Status    string
	Step      uint32
	Timestamp time.Time
}

func (s BaseState) GetStatus() string {
	return s.Status
}

func (s BaseState) GetStep() uint32 {
	return s.Step
}

func (s BaseState) GetTimestamp() time.Time {
	return s.Timestamp
}

// Environment defines the basic interface any environment must implement
type Environment[A agent.Agent, S State] interface {
	// GetState returns the current environment state
	GetState() S
	// Reset resets the environment to initial conditions
	Reset() error
	// AddAgent registers a new agent in the environment
	AddAgent(agent A) error
	// RemoveAgent removes an agent from the environment
	RemoveAgent(agent A) error
	// GetAgents returns all agents in the environment
	GetAgents() []A
	// Step advances the environment by one timestep
	Step(ctx context.Context) error
}

// BaseEnvironment provides common environment functionality
type BaseEnvironment[A agent.Agent, S State] struct {
	agents []A
	state  S
	mu     sync.RWMutex
}

func NewBaseEnvironment[A agent.Agent, S State](initialState S) *BaseEnvironment[A, S] {
	return &BaseEnvironment[A, S]{
		agents: make([]A, 0),
		state:  initialState,
	}
}

func (e *BaseEnvironment[A, S]) GetState() S {
	e.mu.RLock()
	defer e.mu.RUnlock()
	return e.state
}

func (e *BaseEnvironment[A, S]) AddAgent(agent A) error {
	e.mu.Lock()
	defer e.mu.Unlock()
	e.agents = append(e.agents, agent)
	return nil
}

func (e *BaseEnvironment[A, S]) RemoveAgent(agent A) error {
	e.mu.Lock()
	defer e.mu.Unlock()
	for i, a := range e.agents {
		if a.GetID() == agent.GetID() {
			e.agents = append(e.agents[:i], e.agents[i+1:]...)
			return nil
		}
	}
	return fmt.Errorf("agent not found")
}

func (e *BaseEnvironment[A, S]) GetAgents() []A {
	e.mu.RLock()
	defer e.mu.RUnlock()
	return e.agents
}

func (e *BaseEnvironment[A, S]) Reset() error {
	e.mu.Lock()
	defer e.mu.Unlock()
	e.agents = make([]A, 0)
	return nil
}

// Step provides basic step functionality - derived environments should override this
func (e *BaseEnvironment[A, S]) Step(ctx context.Context) error {
	e.mu.Lock()
	defer e.mu.Unlock()

	var wg sync.WaitGroup
	for _, a := range e.agents {
		wg.Add(1)
		go func(a A) {
			defer wg.Done()
			_, err := a.Run(ctx)
			if err != nil {
				log.Printf("error running agent: %s", err)
			}
		}(a)
	}

	wg.Wait()
	return nil
}
