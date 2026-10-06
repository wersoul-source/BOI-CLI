// Package subagent declares the disabled and gated SubAgent market boundary.
package subagent

import "github.com/boi-family/boi-cli/internal/block"

// Manifest returns the owner-approved responsibility of the SubAgent block.
func Manifest() block.Manifest {
	return block.Manifest{
		ID:      block.SubAgentID,
		Name:    "SubAgent",
		Purpose: "Discover and load bounded SubAgent packages only through the block-specific Skill and gate.",
		Owns: []string{
			"SubAgent Skill", "SubAgent Index", "SubAgent packages", "delegation contracts", "delegated budgets",
		},
		DoesNotOwn: []string{"inherited ambient authority", "automatic loading", "unbounded delegation", "Core Persona"},
		Status:     block.StatusDisabled,
	}
}
