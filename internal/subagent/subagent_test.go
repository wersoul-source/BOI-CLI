package subagent_test

import (
	"context"
	"os"
	"path/filepath"
	"strings"
	"sync"
	"testing"
	"time"

	"github.com/boi-family/boi-cli/internal/core/persona"
	"github.com/boi-family/boi-cli/internal/equipment/tools/filesystem"
	"github.com/boi-family/boi-cli/internal/runtime/agent"
	llm "github.com/boi-family/boi-cli/internal/runtime/llm"
	"github.com/boi-family/boi-cli/internal/runtime/workspace"
	"github.com/boi-family/boi-cli/internal/subagent"
)

// script replays responses in order for the parent and its child, which
// share one Provider, and records every request.
type script struct {
	mu        sync.Mutex
	responses []llm.CompletionResponse
	requests  []llm.CompletionRequest
}

func (s *script) Name() string              { return "script" }
func (s *script) SupportsNativeTools() bool { return true }
func (s *script) Stream(context.Context, llm.CompletionRequest) (<-chan llm.Token, error) {
	return nil, nil
}
func (s *script) Complete(_ context.Context, req llm.CompletionRequest) (*llm.CompletionResponse, error) {
	s.mu.Lock()
	defer s.mu.Unlock()
	s.requests = append(s.requests, req)
	response := llm.CompletionResponse{Content: "out of script"}
	if len(s.responses) > 0 {
		response, s.responses = s.responses[0], s.responses[1:]
	}
	if response.InputTokens == 0 {
		response.InputTokens = 10
	}
	response.Provider, response.Model = "script", "m"
	return &response, nil
}

func call(name, arguments string) llm.CompletionResponse {
	return llm.CompletionResponse{ToolCalls: []llm.ToolCallRequest{{Name: name, Arguments: arguments}}}
}

func text(content string) llm.CompletionResponse { return llm.CompletionResponse{Content: content} }

func parentService(t *testing.T, provider *script) (*agent.Service, string) {
	t.Helper()
	sandbox, err := workspace.NewSandbox(t.TempDir())
	if err != nil {
		t.Fatal(err)
	}
	if err := os.WriteFile(filepath.Join(sandbox.Root(), "a.txt"), []byte("alpha"), 0o600); err != nil {
		t.Fatal(err)
	}
	service := agent.NewService(&persona.Persona{Name: "test", SystemPrompt: "persona"}, llm.NewRouter([]llm.Provider{provider}), nil, sandbox)
	service.SetNativeToolCalling(true)
	var names []string
	for _, tool := range filesystem.Tools(sandbox) {
		if err := service.RegisterTool(tool); err != nil {
			t.Fatal(err)
		}
		names = append(names, tool.Spec().Name)
	}
	delegate := subagent.NewTool()
	delegate.Bind(service)
	if err := service.RegisterTool(delegate); err != nil {
		t.Fatal(err)
	}
	if err := service.SetActiveTools(append(names, agent.DelegateTool)); err != nil {
		t.Fatal(err)
	}
	return service, sandbox.Root()
}

func toolNames(request llm.CompletionRequest) []string {
	var names []string
	for _, definition := range request.Tools {
		names = append(names, definition.Name)
	}
	return names
}

func TestReadOnlyDelegationRunsWithOnlyTheGrantedTools(t *testing.T) {
	provider := &script{responses: []llm.CompletionResponse{
		call("agent__delegate", `{"task":"Read a.txt and say what it holds.","tools":["workspace.read"]}`),
		call("workspace__read", `{"path":"a.txt"}`),
		text("it holds alpha"),
		text("the helper found alpha"),
	}}
	service, _ := parentService(t, provider)

	result, err := service.Run(context.Background(), "PARENT-ONLY-QUERY find out what a.txt holds")
	if err != nil || result.Response != "the helper found alpha" {
		t.Fatalf("read-only delegation must work in non-interactive runs: err=%v result=%+v", err, result)
	}
	if result.Usage.ProviderCalls != 4 || result.Usage.ToolCalls != 2 {
		t.Fatalf("the task must report parent and child use together: %+v", result.Usage)
	}
	if result.Outcome != agent.OutcomeNoEffect || len(result.ChangedPaths) != 0 {
		t.Fatalf("a delegated read changed nothing: outcome=%s paths=%v", result.Outcome, result.ChangedPaths)
	}
	child := provider.requests[1]
	if got := strings.Join(toolNames(child), ","); got != "context__expand,workspace__read" {
		t.Fatalf("the child may see only its granted Tools (and context.expand), got %s", got)
	}
	for _, message := range child.Messages {
		if strings.Contains(message.Content, "PARENT-ONLY-QUERY") {
			t.Fatal("the child must not see the parent's conversation")
		}
	}
	last := provider.requests[3].Messages
	if !strings.Contains(last[len(last)-1].Content, "it holds alpha") {
		t.Fatal("the parent must receive the child's answer as an observation")
	}
}

func TestDelegationRefusesToolsTheParentCannotUse(t *testing.T) {
	for name, tools := range map[string]string{
		"unknown Tool":     `["process.run"]`,
		"nested delegate":  `["agent.delegate"]`,
		"no Tools granted": `[]`,
	} {
		provider := &script{responses: []llm.CompletionResponse{
			call("agent__delegate", `{"task":"x","tools":`+tools+`}`),
			text("gave up"),
		}}
		service, _ := parentService(t, provider)
		result, err := service.Run(context.Background(), "delegate")
		if err != nil || result.Response != "gave up" || result.Usage.ToolCalls != 0 {
			t.Fatalf("%s: the delegation must be refused before any child starts: err=%v result=%+v", name, err, result)
		}
		if len(provider.requests) != 2 {
			t.Fatalf("%s: no child request may be made, got %d requests", name, len(provider.requests))
		}
	}
}

func TestChildCannotCallAToolOutsideItsGrant(t *testing.T) {
	provider := &script{responses: []llm.CompletionResponse{
		call("agent__delegate", `{"task":"write b.txt","tools":["workspace.read"]}`),
		call("workspace__write", `{"path":"b.txt","content":"x"}`),
		call("workspace__write", `{"path":"b.txt","content":"x"}`),
		call("workspace__write", `{"path":"b.txt","content":"x"}`),
		text("could not"),
		text("parent done"),
	}}
	service, root := parentService(t, provider)
	_, _ = service.Run(context.Background(), "try")
	retry := provider.requests[2].Messages
	if !strings.Contains(retry[len(retry)-1].Content, "not in the active registry") {
		t.Fatalf("the child's call outside its grant must be rejected by its own Broker: %s", retry[len(retry)-1].Content)
	}
	if _, err := os.Stat(filepath.Join(root, "b.txt")); !os.IsNotExist(err) {
		t.Fatal("a child must never use a Tool it was not granted")
	}
}

func TestDelegatedWriteIsDeniedWithoutApproval(t *testing.T) {
	provider := &script{responses: []llm.CompletionResponse{
		call("agent__delegate", `{"task":"write b.txt","tools":["workspace.write"]}`),
	}}
	service, root := parentService(t, provider)
	result, err := service.Run(context.Background(), "delegate a write")
	if err == nil || result.StopReason != agent.StopNeedsApproval {
		t.Fatalf("a delegation that grants a write needs approval like the write itself: err=%v result=%+v", err, result)
	}
	if _, statErr := os.Stat(filepath.Join(root, "b.txt")); !os.IsNotExist(statErr) {
		t.Fatal("nothing may be written")
	}
}

func TestParentAndChildShareOneBudget(t *testing.T) {
	expensive := func(response llm.CompletionResponse) llm.CompletionResponse {
		response.InputTokens = 400
		return response
	}
	provider := &script{responses: []llm.CompletionResponse{
		expensive(call("agent__delegate", `{"task":"read","tools":["workspace.read"]}`)),
		expensive(call("workspace__read", `{"path":"a.txt"}`)),
		expensive(call("workspace__read", `{"path":"a.txt"}`)),
		expensive(text("child done")),
		expensive(text("parent done")),
	}}
	service, _ := parentService(t, provider)
	limits := agent.DefaultEngineLimits()
	limits.MaxTokens = 1000
	service.SetLimits(limits)

	result, err := service.Run(context.Background(), "spend")
	if err == nil || result.StopReason == agent.StopCompleted {
		t.Fatalf("parent plus child must stop at the parent's token budget: err=%v result=%+v", err, result)
	}
	total := 0
	for range provider.requests {
		total += 400
	}
	if total > limits.MaxTokens+400 {
		t.Fatalf("spent %d tokens, more than one request past the %d budget", total, limits.MaxTokens)
	}
}

func TestGrantingContextExpandIsHarmless(t *testing.T) {
	provider := &script{responses: []llm.CompletionResponse{
		call("agent__delegate", `{"task":"Read a.txt.","tools":["workspace.read","context.expand"]}`),
		text("child done"),
		text("parent done"),
	}}
	service, _ := parentService(t, provider)
	if err := service.SetActiveTools([]string{"workspace.read", agent.DelegateTool}); err != nil {
		t.Fatal(err)
	}
	result, err := service.Run(context.Background(), "delegate")
	if err != nil || result.Response != "parent done" || result.Usage.ToolCalls != 1 {
		t.Fatalf("granting context.expand must not break delegation: err=%v result=%+v", err, result)
	}
}

// startApproving runs an interactive task and approves every request.
func startApproving(t *testing.T, service *agent.Service, query string) *agent.AgentResult {
	t.Helper()
	var result *agent.AgentResult
	for event := range service.Start(context.Background(), query) {
		if event.Approval != nil {
			event.Approval.Decisions <- agent.ApprovalDecision{RequestID: event.Approval.Request.ID, State: agent.ApprovalApproved, DecidedAt: time.Now()}
		}
		if event.Err != nil {
			t.Fatal(event.Err)
		}
		if event.Result != nil {
			result = event.Result
		}
	}
	if result == nil {
		t.Fatal("no result")
	}
	return result
}

func TestDelegatedChangeCountsOnlyWhenTheChildReallyMadeIt(t *testing.T) {
	provider := &script{responses: []llm.CompletionResponse{
		call("agent__delegate", `{"task":"write b.txt","tools":["workspace.write"]}`),
		call("workspace__write", `{"path":"b.txt","content":"beta"}`),
		text("wrote b.txt"),
		text("the helper wrote b.txt"),
	}}
	service, root := parentService(t, provider)
	result := startApproving(t, service, "delegate a write")
	if _, err := os.Stat(filepath.Join(root, "b.txt")); err != nil {
		t.Fatalf("the child's approved write did not happen: %v", err)
	}
	if result.Outcome != agent.OutcomeChanged || len(result.ChangedPaths) != 1 || result.ChangedPaths[0] != "b.txt" || result.Notice != "" {
		t.Fatalf("the child's real change must reach the parent's outcome: %+v", result)
	}
}

func TestDelegationAloneProvesNothing(t *testing.T) {
	// The child answers "done" without calling any Tool; the parent repeats
	// the claim. The delegate call itself must not count as a change.
	provider := &script{responses: []llm.CompletionResponse{
		call("agent__delegate", `{"task":"write a.txt","tools":["workspace.write"]}`),
		text("done, a.txt is rewritten"),
		text("the helper rewrote a.txt"),
		text("the helper rewrote a.txt"),
	}}
	service, root := parentService(t, provider)
	result := startApproving(t, service, "ask the helper to rewrite a.txt")
	if data, _ := os.ReadFile(filepath.Join(root, "a.txt")); string(data) != "alpha" {
		t.Fatalf("nothing should have been written: %q", data)
	}
	if result.Outcome == agent.OutcomeChanged || len(result.ChangedPaths) != 0 {
		t.Fatalf("a delegation without any change was counted as one: %+v", result)
	}
	if result.Notice == "" {
		t.Fatalf("the user must be told that nothing was changed: %+v", result)
	}
}
