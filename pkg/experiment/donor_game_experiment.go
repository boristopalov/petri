package experiment

import (
	"context"
	"fmt"
	"log"
	"math"
	"os"
	"strings"
	"time"

	"github.com/boristopalov/petri/pkg/agent"
	"github.com/boristopalov/petri/pkg/environment"
)

// DonorGameExperiment runs the donor game with generational evolution
type DonorGameExperiment struct {
	env                 *environment.DonorGameEnvironment
	agentFactory        func(ctx context.Context, id string, strategy string) (*agent.DonorGameAgent, error)
	survivorRatio       float64 // fraction of agents that survive to next generation
	numAgents           int     // number of agents per generation
	numGenerations      int
	roundsPerGeneration int
	statsFile           *os.File // file for logging statistics
}

// NewDonorGameExperiment creates a new donor game experiment
func NewDonorGameExperiment(
	env *environment.DonorGameEnvironment,
	agentFactory func(ctx context.Context, id string, strategy string) (*agent.DonorGameAgent, error),
	survivorRatio float64,
	numAgents int,
	numGenerations int,
	roundsPerGeneration int,
) (*DonorGameExperiment, error) {
	// Create stats file with timestamp
	timestamp := time.Now().Format("2006-01-02_15-04-05")
	statsFile, err := os.Create(fmt.Sprintf("experiment_stats_%s.csv", timestamp))
	if err != nil {
		log.Printf("Warning: Failed to create stats file: %v", err)
	} else {
		// Write CSV header
		header := "Generation,TotalResources,AverageResources,StandardDeviation,ResourceInequality,SuccessfulDonations,FailedDonations,SuccessRate\n"
		statsFile.WriteString(header)
	}

	return &DonorGameExperiment{
		env:                 env,
		agentFactory:        agentFactory,
		survivorRatio:       survivorRatio,
		numAgents:           numAgents,
		numGenerations:      numGenerations,
		roundsPerGeneration: roundsPerGeneration,
		statsFile:           statsFile,
	}, nil
}

// Run executes the experiment for the specified number of generations
func (e *DonorGameExperiment) Run(ctx context.Context) error {
	// Initialize first generation
	if err := e.initializeGeneration(ctx, 1, ""); err != nil {
		return fmt.Errorf("failed to initialize first generation: %v", err)
	}

	// Run for specified number of generations
	for gen := 1; gen <= e.numGenerations; gen++ {
		log.Printf("Starting generation %d", gen)

		// Run all rounds in this generation
		if err := e.runGeneration(ctx, gen); err != nil {
			return fmt.Errorf("failed to run generation %d: %v", gen, err)
		}

		// Print generation statistics
		e.printGenerationStats(gen)

		// Select survivors and get their strategies
		survivors := e.selectSurvivors()
		survivorAdvice := e.getSurvivorAdvice(survivors)

		// Initialize next generation with survivors' strategies
		if gen < e.numGenerations {
			if err := e.initializeGeneration(ctx, gen+1, survivorAdvice); err != nil {
				return fmt.Errorf("failed to initialize generation %d: %v", gen+1, err)
			}
		}
	}

	// Close stats file
	if e.statsFile != nil {
		e.statsFile.Close()
	}

	return nil
}

// Initialize a new generation of agents
func (e *DonorGameExperiment) initializeGeneration(ctx context.Context, generation int, survivorAdvice string) error {
	log.Printf("Initializing generation %d", generation)

	// Reset environment
	if err := e.env.Reset(); err != nil {
		return err
	}

	// Create agents
	for i := 0; i < e.numAgents; i++ {
		id := fmt.Sprintf("%d_%d", generation, i)
		agent, err := e.agentFactory(ctx, id, "")
		if err != nil {
			return fmt.Errorf("failed to create agent: %v", err)
		}

		// Generate strategy for the agent
		if err := agent.GenerateStrategy(ctx, generation, survivorAdvice); err != nil {
			return fmt.Errorf("failed to generate strategy for agent %s: %v", id, err)
		}

		// Add agent to environment
		if err := e.env.AddAgent(agent); err != nil {
			return fmt.Errorf("failed to add agent to environment: %v", err)
		}
	}

	return nil
}

// Run all rounds in current generation
func (e *DonorGameExperiment) runGeneration(ctx context.Context, generation int) error {
	roundsPerGen := e.env.GetRoundsPerGen()
	for round := 0; round < roundsPerGen; round++ {
		log.Printf("Generation %d, Round %d/%d", generation, round+1, roundsPerGen)
		if err := e.env.Step(ctx); err != nil {
			return err
		}
	}
	return nil
}

// Select top performing agents to survive to next generation
func (e *DonorGameExperiment) selectSurvivors() []string {
	numSurvivors := int(float64(e.numAgents) * e.survivorRatio)
	return e.env.GetTopAgents(numSurvivors)
}

// Get advice from surviving agents for the next generation
func (e *DonorGameExperiment) getSurvivorAdvice(survivors []string) string {
	state := e.env.GetState()
	var advice []string
	for _, id := range survivors {
		resources := state.AgentResources[id]
		for _, agent := range e.env.GetAgents() {
			if agent.GetID() == id {
				advice = append(advice, fmt.Sprintf("Agent %s (%.2f resources): %s",
					id, resources, agent.GetStrategy()))
				break
			}
		}
	}
	return "Successful strategies from previous generation:\n" +
		strings.Join(advice, "\n")
}

// Print statistics for the current generation
func (e *DonorGameExperiment) printGenerationStats(generation int) {
	state := e.env.GetState()

	// Calculate statistics
	var totalResources float64
	var minResources = math.MaxFloat64
	var maxResources = -math.MaxFloat64
	resources := make([]float64, 0, len(state.AgentResources))

	for _, r := range state.AgentResources {
		totalResources += r
		resources = append(resources, r)
		if r < minResources {
			minResources = r
		}
		if r > maxResources {
			maxResources = r
		}
	}

	// Calculate mean
	avgResources := totalResources / float64(len(state.AgentResources))

	// Calculate standard deviation
	var sumSquares float64
	for _, r := range resources {
		diff := r - avgResources
		sumSquares += diff * diff
	}
	stdDev := math.Sqrt(sumSquares / float64(len(resources)))

	resourceInequality := maxResources - minResources

	// Calculate donation success rate
	totalDonations := state.SuccessfulDonations + state.FailedDonations
	var successRate float64
	if totalDonations > 0 {
		successRate = float64(state.SuccessfulDonations) / float64(totalDonations) * 100
	}

	// Print to console
	log.Printf("\n=== Generation %d Statistics ===", generation)
	log.Printf("Resource Metrics:")
	log.Printf("  Total Resources: %.2f", totalResources)
	log.Printf("  Average Resources: %.2f", avgResources)
	log.Printf("  Standard Deviation: %.2f", stdDev)
	log.Printf("  Resource Inequality (max-min): %.2f", resourceInequality)
	log.Printf("\nDonation Metrics:")
	log.Printf("  Successful Donations: %d", state.SuccessfulDonations)
	log.Printf("  Failed Donations: %d", state.FailedDonations)
	log.Printf("  Success Rate: %.1f%%", successRate)
	log.Printf("==========================\n")

	// Log to CSV file
	if e.statsFile != nil {
		csvLine := fmt.Sprintf("%d,%.2f,%.2f,%.2f,%.2f,%d,%d,%.1f\n",
			generation,
			totalResources,
			avgResources,
			stdDev,
			resourceInequality,
			state.SuccessfulDonations,
			state.FailedDonations,
			successRate,
		)
		if _, err := e.statsFile.WriteString(csvLine); err != nil {
			log.Printf("Warning: Failed to write to stats file: %v", err)
		}
	}
}
