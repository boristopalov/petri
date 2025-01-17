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
		Short: "Run an experiment",
	}

	chatCmd := &cobra.Command{
		Use:   "chat",
		Short: "Run a chat room experiment where agents converse with each other",
		RunE:  runChatExperiment,
	}

	donorGameCmd := &cobra.Command{
		Use:   "donor-game",
		Short: "Run a donor game experiment to study the evolution of cooperation",
		RunE:  runDonorGameExperiment,
	}

	// Add flags for donor game
	donorGameCmd.Flags().IntP("generations", "g", 3, "Number of generations to run")
	donorGameCmd.Flags().IntP("rounds", "r", 3, "Number of rounds per generation")
	donorGameCmd.Flags().IntP("num-agents", "n", 6, "Number of agents per generation")
	donorGameCmd.Flags().Float64P("survivor-ratio", "s", 0.5, "Fraction of agents that survive to next generation")
	donorGameCmd.Flags().Float64P("donation-multiplier", "m", 2.0, "Multiplier for donations (recipient gets this times what donor gives)")
	donorGameCmd.Flags().Float64P("initial-balance", "b", 10.0, "Initial resource balance for each agent")
	donorGameCmd.Flags().StringP("model", "l", "gpt-4", "LLM model to use (gpt-4 or gemini)")

	for _, envFile := range []string{
		".env",
		"../../.env",
		"../../../.env",
	} {
		if err := godotenv.Load(envFile); err == nil {
			break
		}
	}

	runCmd.AddCommand(chatCmd, donorGameCmd)
	rootCmd.AddCommand(runCmd)
	rootCmd.Execute()
}

// runChatExperiment runs a simple chat room experiment where agents converse with each other
func runChatExperiment(cmd *cobra.Command, args []string) error {
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
	env := environment.NewBaseEnvironment[*agent.LLMAgent, environment.BaseState](environment.BaseState{
		Status:    "idle",
		Step:      0,
		Timestamp: time.Now(),
	})

	openai, err := providers.OpenAi(ctx)
	if err != nil {
		return err
	}
	// Create 3 agents
	const NUM_AGENTS = 3
	for i := 0; i < NUM_AGENTS; i++ {
		a, err := agent.NewLLMAgent(
			ctx,
			agent.WithMessageBroker(broker),
			agent.WithTask("Have a friendly conversation about artificial intelligence with other agents."),
			agent.WithProvider(openai),
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

func runDonorGameExperiment(cmd *cobra.Command, args []string) error {
	// Get flag values
	numGenerations, _ := cmd.Flags().GetInt("generations")
	roundsPerGen, _ := cmd.Flags().GetInt("rounds")
	numAgents, _ := cmd.Flags().GetInt("num-agents")
	survivorRatio, _ := cmd.Flags().GetFloat64("survivor-ratio")
	donationMult, _ := cmd.Flags().GetFloat64("donation-multiplier")
	initialBalance, _ := cmd.Flags().GetFloat64("initial-balance")
	modelName, _ := cmd.Flags().GetString("model")

	// Setup context with timeout
	ctx, cancel := context.WithTimeout(context.Background(), 1*time.Hour)
	defer cancel()

	// Handle graceful shutdown
	sigChan := make(chan os.Signal, 1)
	signal.Notify(sigChan, os.Interrupt)
	go func() {
		<-sigChan
		cancel()
	}()

	// Create message broker for agent communication
	broker := messaging.NewBroker()
	defer broker.Reset()

	// Create LLM provider based on model flag
	var llmProvider agent.Client
	var err error
	switch modelName {
	case "gpt-4":
		llmProvider, err = providers.OpenAi(ctx)
	case "gemini":
		llmProvider, err = providers.Gemini(ctx)
	default:
		return fmt.Errorf("unsupported model: %s", modelName)
	}
	if err != nil {
		return fmt.Errorf("failed to create LLM provider: %v", err)
	}

	// Create donor game environment
	env := environment.NewDonorGameEnvironment(
		roundsPerGen,
		donationMult,
		initialBalance,
	)

	// Create agent factory for generating new agents
	agentFactory := func(ctx context.Context, id string, strategy string) (*agent.DonorGameAgent, error) {
		return agent.NewDonorGameAgent(
			ctx,
			id,
			strategy,
			agent.WithProvider(llmProvider),
			agent.WithMessageBroker(broker),
		)
	}

	// Create experiment config
	// config := &config.ExperimentConfig{
	// 	Name:  "donor_game_experiment",
	// 	Steps: roundsPerGen,
	// }

	// Create and run the generational experiment
	experiment, err := experiment.NewDonorGameExperiment(
		env,
		agentFactory,
		survivorRatio,
		numAgents,
		numGenerations,
		roundsPerGen,
	)
	if err != nil {
		return fmt.Errorf("failed to create experiment: %v", err)
	}

	// Run the experiment
	if err := experiment.Run(ctx); err != nil {
		return fmt.Errorf("experiment failed: %v", err)
	}

	return nil
}
