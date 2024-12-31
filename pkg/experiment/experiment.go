package experiment

import (
	"context"
	"sync"
	"time"

	"github.com/boristopalov/petri/pkg/config"
	"github.com/boristopalov/petri/pkg/core"
)

type BaseExperiment struct {
	name    string
	agents  []core.Agent
	steps   int
	env     core.Environment
	metrics *Metrics
	mu      sync.RWMutex
	status  core.ExperimentStatus
}

// Logger handles logging for experiments
type Logger struct {
	// TODO: implement logger
}

// Metrics tracks experiment metrics
type Metrics struct {
	// TODO: implement metrics
}

func NewExperiment(experimentParams *config.ExperimentConfig) BaseExperiment {
	return BaseExperiment{
		name:    experimentParams.Name,
		steps:   10,
		env:     nil,
		metrics: nil,
		status: core.ExperimentStatus{
			Running: false,
		},
	}
}

func (e *BaseExperiment) Run(ctx context.Context) error {
	e.mu.Lock()
	e.status.Running = true
	e.status.StartTime = time.Now()
	e.mu.Unlock()

	defer func() {
		e.mu.Lock()
		e.status.Running = false
		e.status.EndTime = time.Now()
		e.mu.Unlock()
	}()

	return e.runLoop(ctx)
}

func (e *BaseExperiment) runLoop(ctx context.Context) error {

	for i := 0; i < e.steps; i++ {
		for {
			select {
			case <-ctx.Done():
				return ctx.Err()
			default:
				if err := e.step(ctx); err != nil {
					return err
				}
			}
		}
	}
	return nil
}

func (e *BaseExperiment) step(ctx context.Context) error {
	// Get current state
	state := e.env.GetState()

	// Collect actions from all agents
	actions := make([]core.Action, 0, len(e.agents))
	for _, agent := range e.agents {
		obs := createObservation(state, agent.GetID())
		action, err := agent.Generate(ctx, obs)
		if err != nil {
			return err
		}
		actions = append(actions, action)
	}

	// Step environment
	observations, err := e.env.Step(ctx, actions)
	if err != nil {
		return err
	}

	// Update agents
	for _, obs := range observations {
		for _, agent := range e.agents {
			if err := agent.Observe(ctx, core.Event{
				Type:      "observation",
				Content:   obs,
				Timestamp: time.Now(),
				AgentID:   agent.GetID(),
			}); err != nil {
				return err
			}
		}
	}

	return nil
}

func createObservation(state core.State, agentID string) core.Observation {
	// TODO: implement observation creation
	return core.Observation{}
}
