package subagent

import "github.com/boi-family/boi-cli/internal/block"

// Manifest returns the owner-approved responsibility of the SubAgent block.
func Manifest() block.Manifest {
	return block.Manifest{
		ID:      block.SubAgentID,
		Name:    "SubAgent",
		Purpose: "Delegate a self-contained sub-task to a child Agent limited to a subset of the parent's Tools (ADR-003).",
		Owns: []string{
			"SubAgent Skill", "SubAgent Index", "SubAgent packages", "delegation contracts", "delegated budgets",
		},
		DoesNotOwn: []string{"inherited ambient authority", "automatic loading", "unbounded delegation", "Core Persona"},
		Status:     block.StatusPartial,
	}
}
