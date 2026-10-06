// Package core declares BOI identity, qualification, calibration, and Agent
// environment composition ownership.
package core

import "github.com/boi-family/boi-cli/internal/block"

// Manifest returns the owner-approved responsibility of the Core block.
func Manifest() block.Manifest {
	return block.Manifest{
		ID:      block.CoreID,
		Name:    "Core",
		Purpose: "Qualify a Provider Model and compose a standards-based BOI Agent environment.",
		Owns: []string{
			"BOI constitution", "single boi Persona", "Agent instance identity",
			"provider capability testing", "capability profile", "environment composition",
		},
		DoesNotOwn: []string{"provider transport", "tool execution", "artifact storage", "ambient SubAgent authority"},
		Status:     block.StatusPartial,
	}
}
