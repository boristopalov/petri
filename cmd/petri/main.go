package main

import (
	"context"
	"fmt"
	"log"
	"os"
	"os/signal"
	"time"

	"github.com/joho/godotenv"

	"github.com/boristopalov/petri/pkg/agent"
	"github.com/boristopalov/petri/pkg/config"
	"github.com/boristopalov/petri/pkg/environment"
	"github.com/boristopalov/petri/pkg/experiment"
	"github.com/boristopalov/petri/pkg/messaging"
	"github.com/boristopalov/petri/pkg/providers"
	"github.com/spf13/cobra"
)

func main() {
	rootCmd := &cobra.Command{
		Use:   "petri",
		Short: "Petri is a tool for running sandboxed AI-AI interaction experiments and for observing emergent cultural behaviors of LLMs.",
	}

	runCmd := &cobra.Command{
		Use:   "run",
		Short: "Run a test experiment with 3 agents",
		RunE:  runExperiment,
	}

	for _, envFile := range []string{
		".env",
		"../../.env",
		"../../../.env",
	} {
		if err := godotenv.Load(envFile); err == nil {
			break
		}
	}

	rootCmd.AddCommand(runCmd)
	rootCmd.Execute()
}

func runExperiment(cmd *cobra.Command, args []string) error {
	broker := messaging.NewBroker()
	defer broker.Reset()
	ctx, cancel := context.WithTimeout(context.Background(), 15*time.Second)
	defer cancel()

	// Handle graceful shutdown
	sigChan := make(chan os.Signal, 1)
	signal.Notify(sigChan, os.Interrupt)
	go func() {
		<-sigChan
		cancel()
	}()

	// Create experiment config
	config := &config.ExperimentConfig{
		Name:  "chat_room",
		Steps: 10, // Run for 10 steps
	}

	// Create base environment
	env := environment.NewBaseEnvironment()

	// Create 3 agents
	const NUM_AGENTS = 3
	for i := 0; i < NUM_AGENTS; i++ {
		a, err := agent.NewLLMAgent(
			ctx,
			agent.WithMessageBroker(broker),
			agent.WithTask("Have a friendly conversation about artificial intelligence with other agents."),
			agent.WithProvider(providers.OpenAi(ctx)),
		)
		if err != nil {
			return fmt.Errorf("failed to create agent: %v", err)
		}
		log.Printf("Created %s", a.GetID())

		// Start message handler for each agent
		a.StartMessageHandler(ctx)

		// Add agent to environment
		if err := env.AddAgent(a); err != nil {
			return fmt.Errorf("failed to add agent to environment: %v", err)
		}
	}

	// Create and run experiment
	exp := experiment.NewBaseExperiment(config, env)

	// Run 5 steps
	for i := 0; i < 5; i++ {
		time.Sleep(1 * time.Second) // avoid rate limiting
		if err := exp.Step(ctx); err != nil {
			return fmt.Errorf("experiment failed: %v", err)
		}
	}

	return nil
}
