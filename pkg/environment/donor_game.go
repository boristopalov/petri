package environment

import (
	"context"
	"fmt"
	"log"
	"math/rand"
	"sort"
	"strings"
	"sync"
	"time"

	"github.com/boristopalov/petri/pkg/agent"
)

// DonorGameState extends State with donor game specific fields
type DonorGameState struct {
	BaseState           State
	Round               int
	TotalRounds         int
	AgentResources      map[string]float64 // maps agent ID to their current resources
	SuccessfulDonations int                // number of successful donations in this generation
	FailedDonations     int                // number of failed donations in this generation
}

// Implement State interface methods
func (s DonorGameState) GetStatus() string {
	return s.BaseState.GetStatus()
}

func (s DonorGameState) GetStep() uint32 {
	return s.BaseState.GetStep()
}

func (s DonorGameState) GetTimestamp() time.Time {
	return s.BaseState.GetTimestamp()
}

// DonorGameEnvironment implements the donor game mechanics
type DonorGameEnvironment struct {
	agents         []*agent.DonorGameAgent
	state          DonorGameState
	roundsPerGen   int
	donationMult   float64 // multiplier for donations (e.g. 2x)
	initialBalance float64
	mu             sync.RWMutex
}

type donation struct {
	donorID     string
	recipientID string
	amount      float64
	err         error
}

// NewDonorGameEnvironment creates a new donor game environment
func NewDonorGameEnvironment(roundsPerGen int, donationMult float64, initialBalance float64) *DonorGameEnvironment {
	initialState := DonorGameState{
		BaseState: BaseState{
			Status:    "idle",
			Step:      0,
			Timestamp: time.Now(),
		},
		Round:               0,
		TotalRounds:         0,
		AgentResources:      make(map[string]float64),
		SuccessfulDonations: 0,
		FailedDonations:     0,
	}

	return &DonorGameEnvironment{
		agents:         make([]*agent.DonorGameAgent, 0),
		state:          initialState,
		roundsPerGen:   roundsPerGen,
		donationMult:   donationMult,
		initialBalance: initialBalance,
	}
}

// AddAgent adds an agent to the environment
func (e *DonorGameEnvironment) AddAgent(agent *agent.DonorGameAgent) error {
	e.mu.Lock()
	defer e.mu.Unlock()

	e.agents = append(e.agents, agent)
	e.state.AgentResources[agent.GetID()] = e.initialBalance
	return nil
}

// RemoveAgent removes an agent from the environment
func (e *DonorGameEnvironment) RemoveAgent(agent *agent.DonorGameAgent) error {
	e.mu.Lock()
	defer e.mu.Unlock()

	for i, a := range e.agents {
		if a.GetID() == agent.GetID() {
			e.agents = append(e.agents[:i], e.agents[i+1:]...)
			delete(e.state.AgentResources, agent.GetID())
			return nil
		}
	}
	return fmt.Errorf("agent %s not found", agent.GetID())
}

// Reset resets the environment for a new generation
func (e *DonorGameEnvironment) Reset() error {
	e.mu.Lock()
	defer e.mu.Unlock()

	// Clear agents
	e.agents = make([]*agent.DonorGameAgent, 0)

	// Reset state but keep generation number
	e.state = DonorGameState{
		BaseState: BaseState{
			Status:    "idle",
			Step:      0,
			Timestamp: time.Now(),
		},
		Round:               0,
		TotalRounds:         0,
		AgentResources:      make(map[string]float64),
		SuccessfulDonations: 0,
		FailedDonations:     0,
	}

	return nil
}

// GetState returns the current state of the environment
func (e *DonorGameEnvironment) GetState() DonorGameState {
	e.mu.RLock()
	defer e.mu.RUnlock()
	return e.state
}

// Step implements one round of the donor game
func (e *DonorGameEnvironment) Step(ctx context.Context) error {
	log.Println("Running Donor Game step")

	// get a copy of agents for shuffling
	e.mu.Lock()
	defer e.mu.Unlock()
	agents := make([]*agent.DonorGameAgent, len(e.agents))
	copy(agents, e.agents)

	if len(agents)%2 != 0 {
		return fmt.Errorf("need even number of agents")
	}

	// Shuffle agents for random pairing
	rand.Shuffle(len(agents), func(i, j int) {
		agents[i], agents[j] = agents[j], agents[i]
	})
	log.Println("Shuffled agents, starting pairs")

	// Channel to collect donations
	donationChan := make(chan donation, len(agents)/2)

	// Launch all donor decisions in parallel
	for i := 0; i < len(agents); i += 2 {
		donor, recipient := agents[i], agents[i+1]
		log.Printf("Created pair: donor %s, recipient %s", donor.GetID(), recipient.GetID())

		// Get recipient's history
		recipientHistory := e.getRecentHistory(recipient.GetID())

		go func(d, r *agent.DonorGameAgent) {
			log.Printf("Running donor %s", d.GetID())
			donationAmount, err := d.MakeDonationDecision(ctx,
				int(e.state.BaseState.GetStep()), // generation
				e.state.Round,
				r.GetID(),
				e.state.AgentResources[r.GetID()],
				recipientHistory,
				e.state.AgentResources[d.GetID()],
			)
			if err != nil {
				donationChan <- donation{
					donorID: d.GetID(),
					err:     fmt.Errorf("donor %s error: %v", d.GetID(), err),
				}
				return
			}

			donationChan <- donation{
				donorID:     d.GetID(),
				recipientID: r.GetID(),
				amount:      donationAmount,
			}
		}(donor, recipient)
	}

	// Collect all donations
	donations := make([]donation, 0, len(agents)/2)
	var errors []error
	for i := 0; i < len(agents)/2; i++ {
		select {
		case <-ctx.Done():
			return ctx.Err()
		case d := <-donationChan:
			if d.err != nil {
				errors = append(errors, d.err)
				e.state.FailedDonations++
				continue
			}
			donations = append(donations, d)
			e.state.SuccessfulDonations++
		}
	}

	if len(errors) > 0 {
		// Log errors but continue with successful donations
		for _, err := range errors {
			log.Printf("Donation error: %v", err)
		}
	}

	// Update round counters
	e.state.Round++
	e.state.TotalRounds++

	// Apply donations and update memories
	for _, d := range donations {
		pctDonation := d.amount / e.state.AgentResources[d.donorID]
		e.state.AgentResources[d.donorID] -= d.amount
		multipliedAmount := d.amount * e.donationMult
		e.state.AgentResources[d.recipientID] += multipliedAmount

		// Update donor's memory
		for _, agent := range e.agents {
			if agent.GetID() == d.donorID {
				donorMemory := fmt.Sprintf("Round: I donated %.2f%% (%.2f) of my resources to %s, leaving me with %.2f resources",
					pctDonation, d.amount, d.recipientID, e.state.AgentResources[d.donorID])
				if err := agent.GetMemory().Store(donorMemory); err != nil {
					log.Printf("Warning: Failed to store memory for donor %s: %v", d.donorID, err)
				}
			}
			if agent.GetID() == d.recipientID {
				recipientMemory := fmt.Sprintf("Round: I received %.2f%% (%.2f multiplied to %.2f) from %s, bringing my resources to %.2f",
					pctDonation, d.amount, multipliedAmount, d.donorID, e.state.AgentResources[d.recipientID])
				if err := agent.GetMemory().Store(recipientMemory); err != nil {
					log.Printf("Warning: Failed to store memory for recipient %s: %v", d.recipientID, err)
				}
				break
			}
		}
	}

	// Check if round needs to reset
	if e.state.Round >= e.roundsPerGen {
		e.state.Round = 0
	}

	return nil
}

// getRecentHistory returns a string describing the recipient's recent interactions
func (e *DonorGameEnvironment) getRecentHistory(agentID string) string {
	memories := make([]string, 0)
	for _, agent := range e.agents {
		if agent.GetID() == agentID {
			allMemories := agent.GetMemory().GetAllMessages()
			// Get up to last 3 interactions
			start := len(allMemories)
			if start > 3 {
				start = 3
			}
			memories = allMemories[len(allMemories)-start:]
			break
		}
	}

	if len(memories) == 0 {
		return "This is the first round, so there is no history of previous interactions."
	}

	return strings.Join(memories, "\n")
}

// InitializeGeneration generates strategies for all agents at the start of a generation
func (e *DonorGameEnvironment) InitializeGeneration(ctx context.Context, generation int, previousGenAdvice string) error {
	for _, agent := range e.agents {
		if err := agent.GenerateStrategy(ctx, generation, previousGenAdvice); err != nil {
			return fmt.Errorf("failed to generate strategy for agent %s: %v", agent.GetID(), err)
		}
	}
	return nil
}

// GetTopAgents returns the IDs of the top performing agents by resources
func (e *DonorGameEnvironment) GetTopAgents(n int) []string {
	state := e.GetState()

	type agentScore struct {
		id        string
		resources float64
	}

	scores := make([]agentScore, 0, len(state.AgentResources))
	for id, resources := range state.AgentResources {
		scores = append(scores, agentScore{id, resources})
	}

	// Sort by resources descending
	sort.Slice(scores, func(i, j int) bool {
		return scores[i].resources > scores[j].resources
	})

	// Get top N agent IDs
	result := make([]string, 0, n)
	for i := 0; i < n && i < len(scores); i++ {
		result = append(result, scores[i].id)
	}

	return result
}

// GetRoundsPerGen returns the number of rounds per generation
func (e *DonorGameEnvironment) GetRoundsPerGen() int {
	return e.roundsPerGen
}

// GetAgents returns a copy of the agents slice
func (e *DonorGameEnvironment) GetAgents() []*agent.DonorGameAgent {
	e.mu.RLock()
	defer e.mu.RUnlock()
	agents := make([]*agent.DonorGameAgent, len(e.agents))
	copy(agents, e.agents)
	return agents
}
