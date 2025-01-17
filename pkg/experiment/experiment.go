package experiment

import (
	"context"
	"log"
	"sync"
	"time"

	"github.com/boristopalov/petri/pkg/agent"
	"github.com/boristopalov/petri/pkg/config"
	"github.com/boristopalov/petri/pkg/environment"
)

// Experiment defines the interface for running experiments
type Experiment[A agent.Agent, S environment.State] interface {
	GetName() string
	GetEnvironment() environment.Environment[A, S]
	Run(ctx context.Context) error
	// Stop gracefully stops the experiment
	Stop() error
	// GetStatus returns current experiment status
	GetStatus() status
	// Steps through
	Step(ctx context.Context) error
}

type status struct {
	Running   bool
	StartTime time.Time
	EndTime   time.Time
	Errors    []error
}

type Metrics interface {
	RecordState(environment.State)
}

type experimentMetrics struct {
	states []environment.State
	mu     sync.RWMutex
}

func NewMetrics() Metrics {
	return &experimentMetrics{
		states: make([]environment.State, 0),
	}
}

// BaseExperiment provides common experiment functionality
type BaseExperiment[A agent.Agent, S environment.State] struct {
	name        string
	environment environment.Environment[A, S]
	startTime   time.Time
	endTime     time.Time
	metrics     Metrics
	config      config.ExperimentConfig
}

func NewBaseExperiment[A agent.Agent, S environment.State](experimentParams *config.ExperimentConfig, env environment.Environment[A, S]) *BaseExperiment[A, S] {
	return &BaseExperiment[A, S]{
		name:        experimentParams.Name,
		environment: env,
		metrics:     NewMetrics(),
	}
}

func (m *experimentMetrics) RecordState(state environment.State) {
	m.mu.Lock()
	defer m.mu.Unlock()
	m.states = append(m.states, state)
}

func (e *BaseExperiment[A, S]) Step(ctx context.Context) error {
	// Record pre-step metrics
	log.Println("Running step...")
	e.metrics.RecordState(e.environment.GetState())

	// Let environment handle the actual simulation step
	if err := e.environment.Step(ctx); err != nil {
		log.Printf("Step failed: %s", err)
		return err
	}

	// Record post-step metrics
	e.metrics.RecordState(e.environment.GetState())

	return nil
}

func (e *BaseExperiment[A, S]) Run(ctx context.Context) error {
	e.startTime = time.Now()
	defer func() {
		e.endTime = time.Now()
	}()

	return e.environment.Step(ctx)
}

func (e *BaseExperiment[A, S]) runLoop(ctx context.Context) error {
	for i := 0; i < e.config.Steps; i++ {
		select {
		case <-ctx.Done():
			return ctx.Err()
		default:
			if err := e.Step(ctx); err != nil {
				log.Printf("Run loop failed: %s", err)
				return err
			}
		}
	}
	return nil
}

func (e *BaseExperiment[A, S]) GetName() string {
	return e.name
}

func (e *BaseExperiment[A, S]) GetEnvironment() environment.Environment[A, S] {
	return e.environment
}
