package block_test

import (
	"go/build"
	"path/filepath"
	"runtime"
	"strings"
	"testing"

	"github.com/boi-family/boi-cli/internal/block"
	"github.com/boi-family/boi-cli/internal/block/agentfolder"
	"github.com/boi-family/boi-cli/internal/block/core"
	"github.com/boi-family/boi-cli/internal/block/equipment"
	runtimeblock "github.com/boi-family/boi-cli/internal/block/runtime"
	"github.com/boi-family/boi-cli/internal/block/service"
	"github.com/boi-family/boi-cli/internal/block/subagent"
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

func TestConcreteBlocksDoNotImportEachOther(t *testing.T) {
	_, filename, _, ok := runtime.Caller(0)
	if !ok {
		t.Fatal("resolve test file path")
	}
	root := filepath.Dir(filename)
	for _, directory := range []string{"service", "core", "equipment", "runtime", "agentfolder", "subagent"} {
		pkg, err := build.Default.ImportDir(filepath.Join(root, directory), build.IgnoreVendor)
		if err != nil {
			t.Fatalf("inspect block %s: %v", directory, err)
		}
		for _, imported := range pkg.Imports {
			if strings.Contains(imported, "/internal/block/") {
				t.Fatalf("block %s directly imports another concrete block: %s", directory, imported)
			}
		}
	}
}
