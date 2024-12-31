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
	"github.com/boristopalov/petri/pkg/messaging"
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
	ctx, cancel := context.WithCancel(context.Background())
	defer cancel()

	// Handle graceful shutdown
	sigChan := make(chan os.Signal, 1)
	signal.Notify(sigChan, os.Interrupt)
	go func() {
		<-sigChan
		cancel()
	}()

	// Create 3 agents
	agents := make([]*agent.LLMAgent, 3)
	for i := range agents {
		agent, err := agent.NewLLMAgent(agent.WithMessageBroker(broker))
		if err != nil {
			return fmt.Errorf("failed to create %s: %v", agent.GetID(), err)
		}
		agents[i] = agent
		log.Printf("Created %s", agent.GetID())
		// Start message handler for each agent
		agent.StartMessageHandler(ctx)
	}

	// Start a goroutine to handle messages for each agent
	for _, a := range agents {
		agent := a // Create a new variable to avoid closure issues
		go func() {
			for {
				select {
				case msg := <-agent.Receive():
					log.Printf("[%s] Received message from %s: %v", agent.GetID(), msg.From, msg.Content)

					// Generate a response using OpenAI
					prompt := fmt.Sprintf("You are %s. Respond to this message from %s: %v",
						agent.GetID(), msg.From, msg.Content)

					response, err := agent.GetClient().Complete(ctx, agent.GetModel().Id, prompt)
					if err != nil {
						log.Printf("[%s] Error generating response: %v", agent.GetID(), err)
						continue
					}

					// Send the response back
					err = agent.Send(messaging.Message{
						Content: response,
						To:      []string{}, // broadcast to all
					})
					if err != nil {
						log.Printf("[%s] Error sending response: %v", agent.GetID(), err)
					}

					// Add a small delay to prevent rate limiting
					time.Sleep(2 * time.Second)
				case <-ctx.Done():
					return
				}
			}
		}()
	}

	// Have agent-1 start a conversation
	startMsg := fmt.Sprintf("Hello everyone! I'm %s. Let's have a conversation about artificial intelligence.", agents[0].GetID())
	err := agents[0].Send(messaging.Message{
		Content: startMsg,
		To:      []string{}, // broadcast to all
	})
	if err != nil {
		return fmt.Errorf("failed to send initial message: %v", err)
	}

	// Let the conversation run for a while
	time.Sleep(10 * time.Second)
	return nil
}
