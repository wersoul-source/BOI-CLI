// Package architecture enforces the Block dependency rules described in
// docs/architecture/BLOCK_ARCHITECTURE.md. It contains tests only.
package architecture

import (
	"go/build"
	"io/fs"
	"path/filepath"
	"runtime"
	"sort"
	"strings"
	"testing"
)

const module = "github.com/boi-family/boi-cli/internal/"

type layer int

const (
	layerVocabulary layer = iota // block, block/port
	layerSpine                   // core, runtime
	layerPlug                    // equipment, service, agentfolder, subagent
	layerPlatform                // platform
	layerApp                     // app
	layerTransport               // transport
	layerTest                    // acceptance, architecture
)

// plugBlocks maps a top-level directory to its Block. Each plug Block may
// only import its own subpackages among plug code.
var plugBlocks = map[string]bool{"equipment": true, "service": true, "agentfolder": true, "subagent": true}

// transportAllowlist records plug-Block imports that Transport still makes
// directly. The target is an empty list: move each one behind app. Adding a
// new entry needs a written reason in the change.
var transportAllowlist = map[string]bool{
	"transport/cli -> equipment/capability":           true,
	"transport/cli -> equipment/memory":               true,
	"transport/cli -> equipment/memory/weight":        true,
	"transport/cli -> equipment/tools/process":        true,
	"transport/cli -> service/config":                 true,
	"transport/cli -> service/config/envfile":         true,
	"transport/cli -> service/provider/catalog":       true,
	"transport/cli -> service/provider/factory":       true,
	"transport/tui -> equipment/capability":           true,
	"transport/tui -> equipment/tools/filesystem":     true,
	"transport/tui -> service/config":                 true,
	"transport/tui -> service/provider/factory":       true,
	"transport/tui/setup -> service/provider/catalog": true,
}

func top(pkg string) string { return strings.SplitN(pkg, "/", 2)[0] }

func classify(pkg string) layer {
	switch top(pkg) {
	case "block":
		return layerVocabulary
	case "core", "runtime":
		return layerSpine
	case "platform":
		return layerPlatform
	case "app":
		return layerApp
	case "transport":
		return layerTransport
	case "acceptance", "architecture":
		return layerTest
	}
	if plugBlocks[top(pkg)] {
		return layerPlug
	}
	return -1
}

func internalImports(t *testing.T) map[string][]string {
	t.Helper()
	_, file, _, ok := runtime.Caller(0)
	if !ok {
		t.Fatal("locate test file")
	}
	root := filepath.Dir(filepath.Dir(file))
	graph := map[string][]string{}
	err := filepath.WalkDir(root, func(path string, d fs.DirEntry, err error) error {
		if err != nil || !d.IsDir() {
			return err
		}
		pkg, importErr := build.Default.ImportDir(path, 0)
		if importErr != nil {
			if _, ok := importErr.(*build.NoGoError); ok {
				return nil
			}
			return importErr
		}
		rel, _ := filepath.Rel(root, path)
		name := filepath.ToSlash(rel)
		for _, imported := range pkg.Imports {
			if strings.HasPrefix(imported, module) {
				graph[name] = append(graph[name], strings.TrimPrefix(imported, module))
			}
		}
		if _, exists := graph[name]; !exists {
			graph[name] = nil
		}
		return nil
	})
	if err != nil {
		t.Fatal(err)
	}
	return graph
}

func TestEveryPackageBelongsToALayer(t *testing.T) {
	for pkg := range internalImports(t) {
		if classify(pkg) < 0 {
			t.Errorf("package %s is outside the Block layout; place it in a Block or update the rules", pkg)
		}
	}
}

func TestBlockDependencyRules(t *testing.T) {
	graph := internalImports(t)
	var stale []string
	used := map[string]bool{}
	for pkg, imports := range graph {
		from := classify(pkg)
		for _, imported := range imports {
			to := classify(imported)
			edge := pkg + " -> " + imported
			switch from {
			case layerVocabulary:
				t.Errorf("%s: vocabulary must import no BOI package", edge)
			case layerSpine:
				if to != layerVocabulary && to != layerSpine && to != layerPlatform {
					t.Errorf("%s: the fixed spine must not depend on a plug Block, app or transport", edge)
				}
			case layerPlug:
				if to == layerPlug && top(imported) != top(pkg) {
					t.Errorf("%s: plug Blocks must not import another plug Block; wire them in app", edge)
				}
				if to == layerApp || to == layerTransport {
					t.Errorf("%s: plug Blocks must not import app or transport", edge)
				}
			case layerPlatform:
				if to != layerPlatform && to != layerVocabulary {
					t.Errorf("%s: platform is OS glue and must not depend on Blocks", edge)
				}
			case layerApp:
				if to == layerTransport {
					t.Errorf("%s: app must not import transport", edge)
				}
			case layerTransport:
				if to == layerPlug {
					if !transportAllowlist[edge] {
						t.Errorf("%s: transport must reach plug Blocks through app", edge)
					}
					used[edge] = true
				}
			}
		}
	}
	for edge := range transportAllowlist {
		if !used[edge] {
			stale = append(stale, edge)
		}
	}
	sort.Strings(stale)
	for _, edge := range stale {
		t.Errorf("allowlist entry %q is no longer used; remove it", edge)
	}
}
