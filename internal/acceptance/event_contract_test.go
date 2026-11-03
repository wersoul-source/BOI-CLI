package acceptance_test

import (
	"os"
	"path/filepath"
	"strings"
	"testing"

	"github.com/boi-family/boi-cli/internal/acceptance/modelzoo"
	"github.com/boi-family/boi-cli/internal/runtime/agent"
	llm "github.com/boi-family/boi-cli/internal/runtime/llm"
	"github.com/boi-family/boi-cli/internal/service/provider/adapters"
)

// readTaskEvents returns the task's event log and its raw bytes.
func readTaskEvents(t *testing.T, root string) (agent.EventLog, string) {
	t.Helper()
	paths, err := filepath.Glob(filepath.Join(root, "agent-folder", "*", "*", "events.jsonl"))
	if err != nil || len(paths) != 1 {
		all, _ := filepath.Glob(filepath.Join(root, "agent-folder", "**"))
		t.Fatalf("expected one event log, found %v (err=%v, folder=%v)", paths, err, all)
	}
	raw, err := os.ReadFile(paths[0])
	if err != nil {
		t.Fatal(err)
	}
	log, err := agent.ReadEvents(strings.NewReader(string(raw)))
	if err != nil {
		t.Fatal(err)
	}
	return log, string(raw)
}

// The Event contract (ARCHITECTURE_IV §5.4, F1, F2), checked on a real task:
// every event has the version, a gapless sequence and host-assigned ids; the
// log tells the story of the task; and it holds metadata only, never the
// Tool's arguments, its output or the file's content (Experience Contract E2).
func TestEventLogHonoursTheContract(t *testing.T) {
	endpoint := modelzoo.Start(t, modelzoo.Native)
	provider := adapters.NewOpenAIProvider("zoo-native", "fixture-key", endpoint.URL, "zoo-model")
	result, root, err := runZooTaskIn(t, []llm.Provider{provider})
	if err != nil || result == nil || !strings.Contains(result.Response, modelzoo.Marker) {
		t.Fatalf("the task did not complete: err=%v result=%s", err, describeResult(result))
	}
	log, raw := readTaskEvents(t, root)
	if len(log.Skipped) != 0 {
		t.Fatalf("a log written by this build has events this build skips: %+v", log.Skipped)
	}
	kinds := map[string]int{}
	for i, event := range log.Events {
		if event.SchemaVersion != agent.EventSchemaVersion || event.Seq != uint64(i+1) {
			t.Fatalf("event %d: version %d, seq %d", i, event.SchemaVersion, event.Seq)
		}
		if event.TaskID == "" || event.AgentID != "agent-"+event.TaskID || event.Time.IsZero() || !agent.KnownEventKind(event.Kind) {
			t.Fatalf("event %d is missing ids or a known kind: %+v", i, event)
		}
		if strings.HasPrefix(event.Kind, "tool.") && event.CallID == "" {
			t.Fatalf("tool event without a call id: %+v", event)
		}
		kinds[event.Kind]++
	}
	for _, kind := range []string{agent.EventTaskStarted, agent.EventToolProposed, agent.EventToolResult, agent.EventTaskStopped} {
		if kinds[kind] == 0 {
			t.Errorf("the log never says %s: %v", kind, kinds)
		}
	}
	if last := log.Events[len(log.Events)-1]; last.Kind != agent.EventTaskStopped || !strings.Contains(string(last.Payload), "completed") {
		t.Errorf("the log does not end with a completed stop: %+v", last)
	}
	for _, secret := range []string{modelzoo.Marker, "README.md", "second line", zooTask} {
		if strings.Contains(raw, secret) {
			t.Errorf("the event log holds content it must not: %q", secret)
		}
	}
	if strings.Contains(raw, "fixture-key") {
		t.Error("the event log holds the API key")
	}
}

// A route change is an event as well as a notice (P1.5, E6).
func TestRouteChangeIsAnEvent(t *testing.T) {
	endpoint := modelzoo.Start(t, modelzoo.RejectsReplay)
	provider := adapters.NewOpenAIProvider("zoo-gateway", "fixture-key", endpoint.URL, "zoo-model")
	result, root, err := runZooTaskIn(t, []llm.Provider{provider})
	if err != nil || result == nil || len(result.Fallbacks) != 1 {
		t.Fatalf("task: err=%v result=%s", err, describeResult(result))
	}
	log, _ := readTaskEvents(t, root)
	found := false
	for _, event := range log.Events {
		if event.Kind == agent.EventRouteChanged {
			found = true
			if !strings.Contains(string(event.Payload), "native→envelope") || !strings.Contains(string(event.Payload), "status 400") {
				t.Errorf("route.changed lacks its route or evidence: %s", event.Payload)
			}
		}
	}
	if !found {
		t.Fatal("the fallback is in the result but not in the event log")
	}
}
