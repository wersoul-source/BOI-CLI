package block_test

import (
	"testing"

	"github.com/boi-family/boi-cli/internal/agentfolder"
	"github.com/boi-family/boi-cli/internal/block"
	"github.com/boi-family/boi-cli/internal/core"
	"github.com/boi-family/boi-cli/internal/equipment"
	runtimeblock "github.com/boi-family/boi-cli/internal/runtime/agent"
	"github.com/boi-family/boi-cli/internal/service"
	"github.com/boi-family/boi-cli/internal/subagent"
)

func TestOwnerApprovedSixBlockManifests(t *testing.T) {
	manifests := []block.Manifest{
		service.Manifest(),
		core.Manifest(),
		equipment.Manifest(),
		runtimeblock.Manifest(),
		agentfolder.Manifest(),
		subagent.Manifest(),
	}

	if len(manifests) != 6 {
		t.Fatalf("manifest count = %d, want 6", len(manifests))
	}

	seen := make(map[block.ID]struct{}, len(manifests))
	for _, manifest := range manifests {
		if err := manifest.Validate(); err != nil {
			t.Fatalf("validate %q manifest: %v", manifest.ID, err)
		}
		if _, exists := seen[manifest.ID]; exists {
			t.Fatalf("duplicate block ID %q", manifest.ID)
		}
		seen[manifest.ID] = struct{}{}
	}
}
