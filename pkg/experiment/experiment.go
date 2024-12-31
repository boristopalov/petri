package experiment

import (
	"context"
	"log"
	"sync"
	"time"

	"github.com/boristopalov/petri/pkg/config"
	"github.com/boristopalov/petri/pkg/environment"
)

type Experiment interface {
	// Run executes the experiment according to configuration
	Run(ctx context.Context) error
	// Stop gracefully stops the experiment
	Stop() error
	// GetStatus returns current experiment status
	GetStatus() status

	Step(ctx context.Context) error
}

type status struct {
	Running   bool
	StartTime time.Time
	EndTime   time.Time
	Errors    []error
}

type BaseExperiment struct {
	name        string
	environment environment.Environment
	config      *config.ExperimentConfig
	metrics     Metrics
	mu          sync.RWMutex
	status      status
}

// Metrics tracks experiment metrics
type Metrics interface {
	RecordState(environment.State)
}

func NewExperiment(experimentParams *config.ExperimentConfig) BaseExperiment {
	return BaseExperiment{
		name:    experimentParams.Name,
		metrics: nil,
		status: status{
			Running: false,
		},
	}
}

func (e *BaseExperiment) Step(ctx context.Context) error {
	// Record pre-step metrics
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
