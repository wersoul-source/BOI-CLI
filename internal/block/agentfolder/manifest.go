// Package agentfolder declares the single tray for temporary Agent material
// and user-facing deliverables.
package agentfolder

import "github.com/boi-family/boi-cli/internal/block"

// Manifest returns the owner-approved responsibility of Agent Folder.
func Manifest() block.Manifest {
	return block.Manifest{
		ID:      block.AgentFolderID,
		Name:    "Agent Folder",
		Purpose: "Keep temporary material and completed deliverables in one predictable task-oriented tray.",
		Owns: []string{
			"bin tray", "output tray", "task directories", "artifact manifests", "retention metadata",
		},
		DoesNotOwn: []string{"tool verification decisions", "Library routing", "workspace-wide file ownership", "secret storage"},
		Status:     block.StatusActive,
	}
}
