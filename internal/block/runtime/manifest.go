// Package runtime declares the execution-engine boundary of BOI Agent Suit.
package runtime

import "github.com/boi-family/boi-cli/internal/block"

// Manifest returns the owner-approved responsibility of the Runtime block.
func Manifest() block.Manifest {
	return block.Manifest{
		ID:      block.RuntimeID,
		Name:    "Runtime",
		Purpose: "Execute a Core-composed Agent environment with bounded lifecycle and authority.",
		Owns: []string{
			"lifecycle", "state machine", "scheduling", "context runtime", "provider routing",
			"capability broker", "approval", "execution", "verification", "recovery",
			"budgets", "cancellation", "events", "checkpoints", "shutdown and resume",
		},
		DoesNotOwn: []string{"Core Persona", "Agent naming", "capability catalog content", "user deliverable classification"},
		Status:     block.StatusPartial,
	}
}
