package agent

import (
	"context"
	"encoding/json"
	"fmt"
	"os"
	"path/filepath"
	"strings"
	"testing"

	"github.com/boi-family/boi-cli/internal/core/persona"
	llm "github.com/boi-family/boi-cli/internal/runtime/llm"
	"github.com/boi-family/boi-cli/internal/runtime/workspace"
)

func numberedLines(count int) string {
	lines := make([]string, count)
	for index := range lines {
		lines[index] = fmt.Sprintf("line %04d ข้อมูลทดสอบ", index+1)
	}
	return strings.Join(lines, "\n")
}

func readOutput(t *testing.T, content string) string {
	t.Helper()
	data, err := json.Marshal(map[string]any{"Path": "big.txt", "Content": content, "Truncated": false})
	if err != nil {
		t.Fatal(err)
	}
	return string(data)
}

func TestLargeObservationKeepsReferenceToExactMiddle(t *testing.T) {
	store := newContextStore()
	content := numberedLines(2000)
	call := &ToolCall{ID: "c1", Tool: "workspace.read"}
	input := DecisionInput{Step: 2, Steps: []AgentStep{{Number: 1, ToolCall: call, ToolResult: &ToolResult{CallID: "c1", Status: ToolSucceeded, Output: readOutput(t, content)}}}}

	observations := collectObservations(input, store)
	shown := observations[0].result.Output
	if len(shown) > maxObservationBytes+512 || !strings.Contains(shown, "context.expand ref=r1") {
		t.Fatalf("large output must be shortened with a reference, got %d bytes", len(shown))
	}
	var decoded map[string]any
	if err := json.Unmarshal([]byte(shown), &decoded); err != nil || decoded["Truncated"] != false || decoded["Path"] != "big.txt" {
		t.Fatalf("other fields must survive shortening: %v %v", err, decoded)
	}

	output, err := contextExpand{}.Execute(withContextStore(context.Background(), store), map[string]any{"ref": "r1", "lines": "1000-1002"})
	if err != nil {
		t.Fatal(err)
	}
	want := "line 1000 ข้อมูลทดสอบ\nline 1001 ข้อมูลทดสอบ\nline 1002 ข้อมูลทดสอบ"
	if !strings.HasSuffix(output.Output, want) || !strings.Contains(output.Output, "verified") {
		t.Fatalf("expand must return the exact original lines, got:\n%s", output.Output)
	}
}

func TestOldObservationsAreReleasedNotLost(t *testing.T) {
	store := newContextStore()
	var steps []AgentStep
	for index := 1; index <= 5; index++ {
		call := &ToolCall{ID: fmt.Sprintf("c%d", index), Tool: "workspace.read"}
		content := strings.Repeat(fmt.Sprintf("file %d body\n", index), 1200)
		steps = append(steps, AgentStep{Number: index, ToolCall: call, ToolResult: &ToolResult{CallID: call.ID, Status: ToolSucceeded, Output: readOutput(t, content)}})
	}
	observations := collectObservations(DecisionInput{Step: 6, Steps: steps}, store)
	if len(observations) != 5 {
		t.Fatalf("every observation keeps a place in context, got %d", len(observations))
	}
	total := 0
	for _, item := range observations {
		total += len(observationJSON(item))
	}
	if total > maxObservationsBytes+maxObservationBytes {
		t.Fatalf("observations exceed the budget: %d bytes", total)
	}
	if !strings.Contains(observations[0].result.Output, "released by host") || !strings.Contains(observations[0].result.Output, "ref=r5") {
		t.Fatalf("the oldest observation must become a stub with its reference: %s", observations[0].result.Output)
	}
	if strings.Contains(observations[4].result.Output, "released by host") {
		t.Fatal("the newest observation must stay in context")
	}
	output, err := contextExpand{}.Execute(withContextStore(context.Background(), store), map[string]any{"ref": "r5", "lines": "1-1"})
	if err != nil || !strings.HasSuffix(output.Output, "file 1 body") {
		t.Fatalf("released observation must be recoverable: %v %q", err, output.Output)
	}
}

func TestContextExpandRejectsBadRequests(t *testing.T) {
	store := newContextStore()
	store.put("c1", "workspace.read", numberedLines(10))
	ctx := withContextStore(context.Background(), store)
	for name, arguments := range map[string]map[string]any{
		"unknown ref":    {"ref": "r9"},
		"bad range":      {"ref": "r1", "lines": "abc"},
		"outside range":  {"ref": "r1", "lines": "20-30"},
		"reversed range": {"ref": "r1", "lines": "5-2"},
	} {
		if _, err := (contextExpand{}).Execute(ctx, arguments); err == nil {
			t.Errorf("%s: expected an error", name)
		}
	}
	if _, err := (contextExpand{}).Execute(context.Background(), map[string]any{"ref": "r1"}); err == nil {
		t.Error("expand without a task store must fail")
	}
	store.mu.Lock()
	entry := store.entries["r1"]
	entry.text += "tampered"
	store.entries["r1"] = entry
	store.mu.Unlock()
	if _, err := (contextExpand{}).Execute(ctx, map[string]any{"ref": "r1"}); err == nil || !strings.Contains(err.Error(), "integrity") {
		t.Errorf("tampered entry must fail integrity: %v", err)
	}
}

func TestAgentExpandsShortenedObservationEndToEnd(t *testing.T) {
	sandbox, err := workspace.NewSandbox(t.TempDir())
	if err != nil {
		t.Fatal(err)
	}
	provider := &scriptedNativeProvider{}
	service := NewService(&persona.Persona{Name: "test", SystemPrompt: "p"}, llm.NewRouter([]llm.Provider{provider}), nil, sandbox)
	service.SetNativeToolCalling(true)
	for _, tool := range builtinTools(sandbox) {
		if err := service.RegisterTool(tool); err != nil {
			t.Fatal(err)
		}
	}
	if err := service.SetActiveTools([]string{"workspace.read"}); err != nil {
		t.Fatal(err)
	}
	if err := writeFile(sandbox.Root(), "big.txt", numberedLines(3000)); err != nil {
		t.Fatal(err)
	}
	provider.responses = []llm.CompletionResponse{
		{ToolCalls: []llm.ToolCallRequest{{Name: "workspace__read", Arguments: `{"path":"big.txt"}`}}},
		{ToolCalls: []llm.ToolCallRequest{{Name: "context__expand", Arguments: `{"ref":"r1","lines":"1500-1500"}`}}},
		{Content: "found"},
	}
	result, err := service.Run(context.Background(), "find line 1500")
	if err != nil || result.Response != "found" {
		t.Fatalf("err=%v result=%+v", err, result)
	}
	last := provider.requests[2].Messages
	if !strings.Contains(last[len(last)-1].Content, "line 1500 ข้อมูลทดสอบ") {
		t.Fatalf("expanded line must reach the model: %s", last[len(last)-1].Content)
	}
}

func writeFile(root, name, content string) error {
	return os.WriteFile(filepath.Join(root, name), []byte(content), 0o600)
}
