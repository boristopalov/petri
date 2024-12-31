package core

import (
	"context"
)

// Environment defines the rules and mechanics of agent interactions
type Environment interface {
	// Step progresses the environment one step, given actions
	Step(ctx context.Context, actions []Action) ([]Observation, error)
	// GetState returns the current environment state
	GetState() State
	// Reset resets the environment to initial conditions
	Reset() error
}

// Experiment coordinates the running of experiments
type Experiment interface {
	// Run executes the experiment according to configuration
	Run(ctx context.Context) error
	// Stop gracefully stops the experiment
	Stop() error
	// GetStatus returns current experiment status
	GetStatus() ExperimentStatus
}
