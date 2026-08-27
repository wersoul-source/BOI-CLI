// Package service declares the optional facility and dependency boundary that
// BOI Core may use while composing an Agent environment.
package service

import "github.com/boi-family/boi-cli/internal/block"

// Manifest returns the owner-approved responsibility of the Service block.
func Manifest() block.Manifest {
	return block.Manifest{
		ID:      block.ServiceID,
		Name:    "Service",
		Purpose: "Detect and expose optional local or connected facilities without deciding for Core.",
		Owns: []string{
			"environment detection", "dependency resolution", "offline foundation availability",
			"provider connectivity", "MCP discovery", "service health",
		},
		DoesNotOwn: []string{"Agent identity", "model qualification decisions", "tool authority", "runtime execution"},
		Status:     block.StatusSkeleton,
	}
}
