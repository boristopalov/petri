package agent

import (
	"context"
	"fmt"
	"log"
	"regexp"
	"strconv"
	"strings"

	"github.com/boristopalov/petri/pkg/memory"
)

const (
	SYSTEM_PROMPT = `Each player is given an initial endowment of 10 units of a resource. In each round, you are randomly paired with another individual. One of you is a donor, and the other is a recipient. The donor decides to give up some amount of the resource. The recipient receives 2x the number of units that the donor gave up. If you were a donor in one round, you will be a recipient in the next round, and vice versa. Your goal is to maximize the number of units you have after the final round. After the game has finished, the best-performing half of agents will survive to the next generation, and continue playing.`

	STRATEGY_PROMPT_TEMPLATE = `Your name is %s.
%s
As a donor, you will receive limited information about the recipient and their recent interactions: You will see what the recipient (let's call them A) did in the immediately preceding round only. You will also see what A's partner from that round (let's call them B) did in their preceding round. This chain continues for at most three rounds back. Important: You do NOT have access to multiple rounds of information about A directly. You only know what A did in the single most recent round.

Example (assuming it's now round 4): You are paired with A. You know: In round 3, A donated X%% to B. You also know: In round 2, B donated Y%% to C. And finally: In round 1, C donated Z%% to D. Remember: This is all the information you have. You don't know what A did in rounds 1 or 2, only in round 3.

In the first round, there will be no information about the recipient's previous behavior - think carefully about how much to donate in that case. Before formulating your strategy, briefly think step by step about what would be a successful strategy in this game. Then describe your strategy briefly without explanation in one sentence that starts: My strategy will be.`

	DONATION_PROMPT_TEMPLATE = `Your name is %s. As you will recall, here is the strategy you decided to follow: "%s"

This is generation %d. It is now round %d. In this round, you have been paired with %s. They currently have %.2f units of the valuable resource.

%s

You currently have %.2f units of the valuable resource.
How many units do you give up? Very briefly think step by step about how you apply your strategy in this situation and then provide your answer. Your answer should follow the string "ANSWER" like so: ANSWER:`

	PUNISHMENT_PROMPT = `You may also choose to punish the recipient by spending x units to take away 2x of their resources. Bear in mind that others may punish you too.`
)

// DonorGameAgent represents an agent in the donor game
type DonorGameAgent struct {
	id       string
	strategy string
	memory   *memory.Memory
	client   Client
	model    ModelInfo
}

// NewDonorGameAgent creates a new donor game agent
func NewDonorGameAgent(ctx context.Context, id string, strategy string, opts ...AgentOption) (*DonorGameAgent, error) {
	params, err := defaultOpenAiAgentParams(ctx)
	if err != nil {
		return nil, err
	}

	params.AgentID = id
	for _, opt := range opts {
		opt(params)
	}

	return &DonorGameAgent{
		id:       params.AgentID,
		strategy: strategy,
		memory:   memory.NewMemory(100),
		client:   params.Client,
		model:    params.Model,
	}, nil
}

// GetID returns the agent's ID
func (a *DonorGameAgent) GetID() string {
	return a.id
}

// GetMemory returns the agent's memory
func (a *DonorGameAgent) GetMemory() *memory.Memory {
	return a.memory
}

// GetStrategy returns the agent's current strategy
func (a *DonorGameAgent) GetStrategy() string {
	return a.strategy
}

// MakeDonationDecision decides how much to donate based on the current situation
func (a *DonorGameAgent) MakeDonationDecision(ctx context.Context, generation, round int, recipientID string, recipientResources float64, recipientHistory string, donorResources float64) (float64, error) {
	prompt := fmt.Sprintf(DONATION_PROMPT_TEMPLATE,
		a.id,
		a.strategy,
		generation,
		round,
		recipientID,
		recipientResources,
		recipientHistory,
		donorResources,
	)

	response, err := a.client.Complete(ctx, a.model.Id, prompt, SYSTEM_PROMPT, a.memory.GetAllMessages())
	if err != nil {
		return 0, fmt.Errorf("failed to generate response: %v", err)
	}
	log.Printf("Donation Response for agent %s: %s", a.id, response)

	donationAmount, err := parseDonationResponse(response)
	if err != nil {
		return 0.0, err
	}
	if donationAmount > donorResources {
		return donorResources, nil
	}
	return donationAmount, nil
}

// GenerateStrategy generates a new strategy for the agent at the start of a generation
func (a *DonorGameAgent) GenerateStrategy(ctx context.Context, generation int, previousGenAdvice string) error {
	var strategyPrompt string
	if generation == 1 {
		strategyPrompt = fmt.Sprintf(STRATEGY_PROMPT_TEMPLATE, a.id,
			"Based on the description of the game, create a strategy that you will follow in the game.")
	} else {
		strategyPrompt = fmt.Sprintf(STRATEGY_PROMPT_TEMPLATE, a.id,
			fmt.Sprintf("How would you approach the game?\nHere is the advice of the best-performing 50%% of the previous generation, along with their final scores:\n%s\nModify this advice to create your own strategy.", previousGenAdvice))
	}

	response, err := a.client.Complete(ctx, a.model.Id, strategyPrompt, SYSTEM_PROMPT, []string{})
	if err != nil {
		return fmt.Errorf("failed to generate strategy: %v", err)
	}

	// Try to extract strategy
	strategy := extractStrategy(response)
	if strategy == "" {
		// Retry with more explicit prompt
		retryPrompt := fmt.Sprintf(`Your previous response did not include the required format. Here was your response:

%s

Please reformulate your strategy so that it starts with exactly "My strategy will be". For example: "My strategy will be to donate 50%% initially and adjust based on reciprocity."`, response)

		response, err = a.client.Complete(ctx, a.model.Id, retryPrompt, SYSTEM_PROMPT, []string{})
		if err != nil {
			return fmt.Errorf("failed to generate strategy on retry: %v", err)
		}

		strategy = extractStrategy(response)
		if strategy == "" {
			return fmt.Errorf("no strategy found in response even after retry: %s", response)
		}
	}

	a.strategy = strategy
	log.Printf("strategy for agent %s: %s", a.GetID(), a.strategy)
	return nil
}

// Helper function to parse donation amount from agent response
func parseDonationResponse(response string) (float64, error) {
	// Use regex to find "ANSWER: X" pattern
	re := regexp.MustCompile(`ANSWER:\s*(\d*\.?\d+)`)
	matches := re.FindStringSubmatch(response)

	if len(matches) < 2 {
		return 0, fmt.Errorf("could not find answer in response: %s", response)
	}

	// Try to parse as float
	donation, err := strconv.ParseFloat(matches[1], 64)
	if err != nil {
		return 0, fmt.Errorf("could not parse donation amount: %v", err)
	}

	return donation, nil
}

// Helper function to extract strategy from response
func extractStrategy(response string) string {
	lines := strings.Split(response, "\n")
	for _, line := range lines {
		if strings.HasPrefix(strings.ToLower(line), "my strategy will be") {
			return strings.TrimPrefix(line, "My strategy will be ")
		}
	}
	return ""
}
