package environment

import (
	"context"
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
	// Step advances the environment by one timestep
	Step(ctx context.Context) error
}
