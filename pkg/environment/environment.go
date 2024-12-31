package environment

import (
	"context"
	"fmt"
	"log"
	"sync"
	"time"

	"github.com/boristopalov/petri/pkg/agent"
)

type State struct {
	status    string
	step      uint32
	timestamp time.Time
}

// Environment defines the rules and mechanics of agent interactions
type Environment interface {
	// GetState returns the current environment state
	GetState() State
	// Reset resets the environment to initial conditions
	Reset() error
	// AddAgent registers a new agent in the environment
	AddAgent(agent agent.Agent) error
	// RemoveAgent removes an agent from the environment
	RemoveAgent(agent agent.Agent) error
	// Gets the list of agents in the env
	GetAgents(agent agent.Agent) ([]agent.Agent, error)
	// Step advances the environment by one timestep
	Step(ctx context.Context) error
}

type BaseEnvironment struct {
	agents []agent.Agent
	state  State
}

func NewBaseEnvironment() *BaseEnvironment {
	return &BaseEnvironment{
		agents: make([]agent.Agent, 0),
		state: State{
			status:    "idle",
			step:      0,
			timestamp: time.Now(),
		},
	}
}

func (e *BaseEnvironment) GetState() State {
	return e.state
}

func (e *BaseEnvironment) Step(ctx context.Context) error {
	// state := e.GetState()
	e.state.status = "running"
	e.state.step++
	e.state.timestamp = time.Now()

	// Wait for each agent to run
	var wg sync.WaitGroup
	for _, a := range e.agents {
		wg.Add(1)
		go func(a agent.Agent) {
			defer wg.Done()
			_, err := a.Run(ctx)
			if err != nil {
				log.Printf("error running agent: %s", err)
			}
		}(a)
		// Process the action and update state accordingly
		// ...
	}

	wg.Wait()
	return nil
}

func (e *BaseEnvironment) AddAgent(agent agent.Agent) error {
	e.agents = append(e.agents, agent)
	return nil
}

func (e *BaseEnvironment) RemoveAgent(agent agent.Agent) error {
	for i, a := range e.agents {
		if a == agent {
			e.agents = append(e.agents[:i], e.agents[i+1:]...)
			return nil
		}
	}
	return fmt.Errorf("agent not found")
}

func (e *BaseEnvironment) GetAgents() []agent.Agent {
	return e.agents
}

func (e *BaseEnvironment) Reset() error {
	e.agents = make([]agent.Agent, 0)
	e.state = State{
		status:    "idle",
		step:      0,
		timestamp: time.Now(),
	}
	return nil
}
