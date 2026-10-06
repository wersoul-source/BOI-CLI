// Package equipment declares the inventory boundary for optional Agent
// equipment such as Skills, Tools, Memory, MCP, Planner, and UI adapters.
package equipment

import "github.com/boi-family/boi-cli/internal/block"

// Manifest returns the owner-approved responsibility of Various Equipment.
func Manifest() block.Manifest {
	return block.Manifest{
		ID:      block.EquipmentID,
		Name:    "Various Equipment",
		Purpose: "Provide discoverable Agent equipment without activating it or granting authority.",
		Owns: []string{
			"MCP", "Tools", "Plugins", "Skills", "Memory", "Knowledge",
			"Commands", "Planner", "Status", "TUI and GUI onboarding equipment",
		},
		DoesNotOwn: []string{"equipment selection policy", "approval", "execution lifecycle", "Core Persona"},
		Status:     block.StatusPartial,
	}
}
