package agentfolder

import (
	"encoding/json"
	"os"
	"path/filepath"
	"strings"
	"testing"
	"time"

	"github.com/boi-family/boi-cli/internal/agent"
)

func TestStoreCreatesOnlyPrimaryTrays(t *testing.T) {
	store, err := NewStore(filepath.Join(t.TempDir(), "agent-folder"))
	if err != nil {
		t.Fatal(err)
	}
	entries, err := os.ReadDir(store.Root())
	if err != nil {
		t.Fatal(err)
	}
	if len(entries) != 2 || entries[0].Name() != "bin" || entries[1].Name() != "output" {
		t.Fatalf("primary trays = %#v", entries)
	}
}

func TestTaskIDCollisionGetsDeterministicSuffix(t *testing.T) {
	store, err := NewStore(filepath.Join(t.TempDir(), "agent-folder"))
	if err != nil {
		t.Fatal(err)
	}
	store.now = func() time.Time { return time.Date(2026, 9, 3, 12, 0, 0, 0, time.UTC) }
	plan := agent.NewPlanner().Plan("same task")
	first, err := store.Begin("same task", plan)
	if err != nil {
		t.Fatal(err)
	}
	second, err := store.Begin("same task", plan)
	if err != nil {
		t.Fatal(err)
	}
	if first.ID == second.ID || second.ID != first.ID+"-02" {
		t.Fatalf("Task IDs = %q, %q", first.ID, second.ID)
	}
}

func TestCompletedTaskProducesDiscoverableSecretFreeManifest(t *testing.T) {
	store, err := NewStore(filepath.Join(t.TempDir(), "agent-folder"))
	if err != nil {
		t.Fatal(err)
	}
	now := time.Date(2026, 9, 3, 12, 0, 0, 0, time.UTC)
	store.now = func() time.Time { return now }
	plan := agent.NewPlanner().Plan("task contains SECRET-VALUE")
	session, err := store.Begin("task contains SECRET-VALUE", plan)
	if err != nil {
		t.Fatal(err)
	}
	if err := os.WriteFile(filepath.Join(session.OutputDir, "report.txt"), []byte("deliverable"), 0o600); err != nil {
		t.Fatal(err)
	}
	result := &agent.AgentResult{
		TaskID: session.ID, Response: "model response SECRET-VALUE", Steps: 2,
		StopReason: agent.StopCompleted, Provider: "provider", Model: "model",
		ProviderProfileRef: ".boi/provider-profiles/profile.json", Plan: plan,
		Usage: agent.Usage{InputTokens: 3, OutputTokens: 4, ProviderCalls: 1, ToolCalls: 1},
		Trace: []agent.AgentStep{{ToolCall: &agent.ToolCall{Tool: "workspace.write"}, ToolResult: &agent.ToolResult{CallID: "call-1", Status: agent.ToolSucceeded}}},
	}
	if err := store.Finalize(session, result); err != nil {
		t.Fatal(err)
	}
	manifestPath := filepath.Join(session.OutputDir, "manifest.json")
	data, err := os.ReadFile(manifestPath)
	if err != nil {
		t.Fatal(err)
	}
	if strings.Contains(string(data), "SECRET-VALUE") {
		t.Fatal("manifest persisted prompt or model output")
	}
	var manifest TaskManifest
	if err := json.Unmarshal(data, &manifest); err != nil {
		t.Fatal(err)
	}
	if manifest.Outcome != "completed" || manifest.ProviderProfileRef == "" || len(manifest.Artifacts) != 1 || len(manifest.Evidence) != 2 {
		t.Fatalf("manifest = %#v", manifest)
	}
	if result.Manifest != filepath.ToSlash(filepath.Join("agent-folder", "output", session.ID, "manifest.json")) || len(result.Artifacts) != 1 {
		t.Fatalf("result references = %#v", result)
	}
}

func TestFailedTaskManifestStaysInBin(t *testing.T) {
	store, err := NewStore(filepath.Join(t.TempDir(), "agent-folder"))
	if err != nil {
		t.Fatal(err)
	}
	plan := agent.NewPlanner().Plan("fail safely")
	session, err := store.Begin("fail safely", plan)
	if err != nil {
		t.Fatal(err)
	}
	result := &agent.AgentResult{TaskID: session.ID, StopReason: agent.StopVerificationFailed, Plan: plan}
	if err := store.Finalize(session, result); err != nil {
		t.Fatal(err)
	}
	if _, err := os.Stat(filepath.Join(session.BinDir, "manifest.json")); err != nil {
		t.Fatalf("failed manifest missing: %v", err)
	}
	if _, err := os.Stat(filepath.Join(session.OutputDir, "manifest.json")); !os.IsNotExist(err) {
		t.Fatalf("failed task appeared as completed output: %v", err)
	}
}

func TestBinCleanupNeverTouchesOutput(t *testing.T) {
	store, err := NewStore(filepath.Join(t.TempDir(), "agent-folder"))
	if err != nil {
		t.Fatal(err)
	}
	plan := agent.NewPlanner().Plan("cleanup")
	session, err := store.Begin("cleanup", plan)
	if err != nil {
		t.Fatal(err)
	}
	old := time.Now().Add(-48 * time.Hour)
	if err := os.Chtimes(session.BinDir, old, old); err != nil {
		t.Fatal(err)
	}
	candidates, err := store.CleanupBinBefore(time.Now().Add(-24*time.Hour), false)
	if err != nil || len(candidates) != 1 {
		t.Fatalf("dry-run candidates=%v err=%v", candidates, err)
	}
	if _, err := os.Stat(session.BinDir); err != nil {
		t.Fatalf("dry-run removed bin: %v", err)
	}
	if _, err := store.CleanupBinBefore(time.Now().Add(-24*time.Hour), true); err != nil {
		t.Fatal(err)
	}
	if _, err := os.Stat(session.BinDir); !os.IsNotExist(err) {
		t.Fatalf("bin task was not removed: %v", err)
	}
	if _, err := os.Stat(session.OutputDir); err != nil {
		t.Fatalf("output was touched by bin cleanup: %v", err)
	}
}
