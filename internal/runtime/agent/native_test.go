package agent

import (
	"context"
	"errors"
	"os"
	"path/filepath"
	"strings"
	"sync"
	"testing"
	"time"

	"github.com/boi-family/boi-cli/internal/block/port"
	"github.com/boi-family/boi-cli/internal/core/persona"
	llm "github.com/boi-family/boi-cli/internal/runtime/llm"
	"github.com/boi-family/boi-cli/internal/runtime/workspace"
)

// scriptedNativeProvider replays one response per call and records requests.
type scriptedNativeProvider struct {
	mu        sync.Mutex
	responses []llm.CompletionResponse
	requests  []llm.CompletionRequest
}

func (p *scriptedNativeProvider) Name() string              { return "native" }
func (p *scriptedNativeProvider) SupportsNativeTools() bool { return true }
func (p *scriptedNativeProvider) Stream(context.Context, llm.CompletionRequest) (<-chan llm.Token, error) {
	return nil, nil
}
func (p *scriptedNativeProvider) Complete(_ context.Context, req llm.CompletionRequest) (*llm.CompletionResponse, error) {
	p.mu.Lock()
	defer p.mu.Unlock()
	p.requests = append(p.requests, req)
	response := llm.CompletionResponse{Content: "done"}
	if len(p.responses) > 0 {
		response, p.responses = p.responses[0], p.responses[1:]
	}
	response.Provider, response.Model = "native", "native-model"
	return &response, nil
}

func nativeService(t *testing.T, provider llm.Provider) (*Service, string) {
	t.Helper()
	sandbox, err := workspace.NewSandbox(t.TempDir())
	if err != nil {
		t.Fatal(err)
	}
	service := withWorkspaceTools(t, NewService(&persona.Persona{Name: "test", SystemPrompt: "persona"}, llm.NewRouter([]llm.Provider{provider}), nil, sandbox), sandbox)
	service.SetNativeToolCalling(true)
	return service, sandbox.Root()
}

func TestNativeBatchReadsManyFilesInOneTurn(t *testing.T) {
	provider := &scriptedNativeProvider{responses: []llm.CompletionResponse{
		{ToolCalls: []llm.ToolCallRequest{
			{ID: "call_0", Name: "workspace__read", Arguments: `{"path":"a.txt"}`},
			{ID: "call_1", Name: "workspace__read", Arguments: `{"path":"b.txt"}`},
			{ID: "call_2", Name: "workspace__read", Arguments: `{"path":"c.txt"}`},
		}},
		{Content: "A B C"},
	}}
	service, root := nativeService(t, provider)
	for name, body := range map[string]string{"a.txt": "alpha", "b.txt": "bravo", "c.txt": "charlie"} {
		if err := os.WriteFile(filepath.Join(root, name), []byte(body), 0o600); err != nil {
			t.Fatal(err)
		}
	}

	result, err := service.Run(context.Background(), "read a, b and c")
	if err != nil {
		t.Fatal(err)
	}
	if result.Response != "A B C" || result.Usage.ProviderCalls != 2 || result.Usage.ToolCalls != 3 {
		t.Fatalf("result=%q usage=%+v", result.Response, result.Usage)
	}
	first := provider.requests[0]
	if len(first.Tools) != 6 || strings.Contains(first.Messages[0].Content, "<boi-action>") {
		t.Fatalf("native request must carry tool definitions instead of envelope instructions: %d tools", len(first.Tools))
	}
	if strings.Contains(first.Messages[0].Content, "Host Task Plan") {
		t.Fatal("the static Task Plan must not be sent to the model")
	}

	// The second request replays the batch as one assistant message whose
	// every call has a matching tool reply.
	second := provider.requests[1].Messages
	var assistant *llm.Message
	replies := map[string]string{}
	for index := range second {
		message := second[index]
		if message.Role == "assistant" && len(message.ToolCalls) > 0 {
			if assistant != nil {
				t.Fatal("one batch must be one assistant message")
			}
			assistant = &second[index]
		}
		if message.Role == "tool" {
			replies[message.ToolCallID] = message.Content
		}
	}
	if assistant == nil || len(assistant.ToolCalls) != 3 {
		t.Fatalf("assistant tool calls missing: %+v", second)
	}
	for index, call := range assistant.ToolCalls {
		reply, ok := replies[call.ID]
		if !ok || call.Name != "workspace__read" {
			t.Fatalf("call %+v has no tool reply", call)
		}
		if want := []string{"alpha", "bravo", "charlie"}[index]; !strings.Contains(reply, want) {
			t.Fatalf("reply for %s = %s, want %s", call.ID, reply, want)
		}
		if call.ID == "call_0" || call.ID == "call_1" {
			t.Fatal("provider call IDs must be replaced by host IDs")
		}
	}
}

func TestNativeBatchWithWriteStopsForApprovalBeforeLaterCalls(t *testing.T) {
	provider := &scriptedNativeProvider{responses: []llm.CompletionResponse{
		{ToolCalls: []llm.ToolCallRequest{
			{ID: "1", Name: "workspace__write", Arguments: `{"path":"out.txt","content":"x"}`},
			{ID: "2", Name: "workspace__read", Arguments: `{"path":"out.txt"}`},
		}},
	}}
	service, root := nativeService(t, provider)

	result, err := service.Run(context.Background(), "write then read")
	if err == nil || result.StopReason != StopNeedsApproval {
		t.Fatalf("a write in a batch must still require approval: err=%v result=%+v", err, result)
	}
	if result.Usage.ToolCalls != 1 {
		t.Fatalf("calls after a denied write must not run: %+v", result.Usage)
	}
	if _, statErr := os.Stat(filepath.Join(root, "out.txt")); !os.IsNotExist(statErr) {
		t.Fatal("unapproved write reached the filesystem")
	}
}

func TestNativeDecisionRejectsUnknownToolAndOversizedBatch(t *testing.T) {
	names := map[string]string{"workspace__read": "workspace.read"}
	if _, err := nativeDecision([]llm.ToolCallRequest{{Name: "shell"}}, names, "p"); err == nil {
		t.Fatal("unknown native tool must be rejected")
	}
	if _, err := nativeDecision([]llm.ToolCallRequest{{Name: "workspace__read", Arguments: "not json"}}, names, "p"); err == nil {
		t.Fatal("malformed arguments must be rejected")
	}
	batch := make([]llm.ToolCallRequest, MaxBatchCalls+1)
	for index := range batch {
		batch[index] = llm.ToolCallRequest{Name: "workspace__read", Arguments: `{"path":"a"}`}
	}
	if _, err := nativeDecision(batch, names, "p"); err == nil {
		t.Fatal("a batch above MaxBatchCalls must be rejected")
	}
}

func TestTextEnvelopeStillWorksWhenNativeIsEnabled(t *testing.T) {
	provider := &scriptedNativeProvider{responses: []llm.CompletionResponse{
		{Content: `<boi-action>{"id":"r","tool":"workspace.list","purpose":"look","arguments":{"path":"."}}</boi-action>`},
		{Content: "listed"},
	}}
	service, _ := nativeService(t, provider)
	result, err := service.Run(context.Background(), "list")
	if err != nil || result.Response != "listed" || result.Usage.ToolCalls != 1 {
		t.Fatalf("envelope fallback failed: err=%v result=%+v", err, result)
	}
}

func TestNativeModeNeedsEveryRouterCandidate(t *testing.T) {
	native := &scriptedNativeProvider{}
	text := &captureProvider{}
	router := llm.NewRouter([]llm.Provider{native, text})
	if router.SupportsNativeTools() {
		t.Fatal("a Router with a text-only candidate must not use native tools")
	}
}

// barrierTool succeeds only when all of its calls run at the same time.
type barrierTool struct {
	name    string
	arrived chan struct{}
	want    int
}

func (b *barrierTool) Spec() port.ToolSpec {
	return port.ToolSpec{Name: b.name, Usage: b.name + "(id)", Risk: port.RiskRead, Approval: port.ApprovalAuto, Timeout: 2 * time.Second}
}
func (b *barrierTool) Describe(map[string]any) port.ToolPreview { return port.ToolPreview{} }
func (b *barrierTool) Execute(ctx context.Context, _ map[string]any) (port.ToolOutput, error) {
	b.arrived <- struct{}{}
	deadline := time.After(time.Second)
	for len(b.arrived) < b.want {
		select {
		case <-deadline:
			return port.ToolOutput{}, errors.New("reads did not run concurrently")
		case <-time.After(time.Millisecond):
		}
	}
	return port.ToolOutput{Output: "ok"}, nil
}

func TestNativeReadBatchRunsConcurrently(t *testing.T) {
	provider := &scriptedNativeProvider{responses: []llm.CompletionResponse{
		{ToolCalls: []llm.ToolCallRequest{
			{Name: "barrier", Arguments: `{"id":"1"}`},
			{Name: "barrier", Arguments: `{"id":"2"}`},
			{Name: "barrier", Arguments: `{"id":"3"}`},
		}},
		{Content: "all read"},
	}}
	service, _ := nativeService(t, provider)
	barrier := &barrierTool{name: "barrier", arrived: make(chan struct{}, 3), want: 3}
	if err := service.RegisterTool(barrier); err != nil {
		t.Fatal(err)
	}
	if err := service.SetActiveTools(service.RegisteredTools()); err != nil {
		t.Fatal(err)
	}
	result, err := service.Run(context.Background(), "read three")
	if err != nil || result.Usage.ToolCalls != 3 {
		t.Fatalf("parallel read batch failed: err=%v result=%+v", err, result)
	}
}

// replayRejectingProvider accepts tool definitions but returns HTTP 400 for
// any request that replays tool messages, like some hosted endpoints.
type replayRejectingProvider struct{ scriptedNativeProvider }

func (p *replayRejectingProvider) Complete(ctx context.Context, req llm.CompletionRequest) (*llm.CompletionResponse, error) {
	for _, message := range req.Messages {
		if message.Role == "tool" {
			return nil, llm.NewHTTPError("native", 400, "unsupported role tool", nil)
		}
	}
	return p.scriptedNativeProvider.Complete(ctx, req)
}

func TestNativeReplayRejectionFallsBackToEnvelopes(t *testing.T) {
	provider := &replayRejectingProvider{scriptedNativeProvider{responses: []llm.CompletionResponse{
		{ToolCalls: []llm.ToolCallRequest{{Name: "workspace__read", Arguments: `{"path":"a.txt"}`}}},
		{Content: "alpha seen"},
	}}}
	service, root := nativeService(t, provider)
	if err := os.WriteFile(filepath.Join(root, "a.txt"), []byte("alpha"), 0o600); err != nil {
		t.Fatal(err)
	}
	result, err := service.Run(context.Background(), "read a")
	if err != nil || result.Response != "alpha seen" {
		t.Fatalf("fallback failed: err=%v result=%+v", err, result)
	}
	last := provider.requests[len(provider.requests)-1]
	if len(last.Tools) != 0 || !strings.Contains(last.Messages[0].Content, "<boi-action>") {
		t.Fatal("the retry must use text envelopes without tool definitions")
	}
	if !strings.Contains(last.Messages[len(last.Messages)-1].Content, "alpha") {
		t.Fatal("the observation must reach the model as text")
	}
}

func TestInvalidNativeArgumentsAreRejectedBeforeAuthorization(t *testing.T) {
	provider := &scriptedNativeProvider{responses: []llm.CompletionResponse{
		{ToolCalls: []llm.ToolCallRequest{{Name: "workspace__write", Arguments: `{"path":"out.txt"}`}}},
		{Content: "gave up"},
	}}
	service, root := nativeService(t, provider)
	result, err := service.Run(context.Background(), "write")
	if err != nil || result.Response != "gave up" || result.Usage.ToolCalls != 0 {
		t.Fatalf("invalid arguments must never reach authorization: err=%v result=%+v", err, result)
	}
	retry := provider.requests[1].Messages
	if !strings.Contains(retry[len(retry)-1].Content, "arguments.content is required") {
		t.Fatalf("the model must be told what was wrong: %s", retry[len(retry)-1].Content)
	}
	if _, statErr := os.Stat(filepath.Join(root, "out.txt")); !os.IsNotExist(statErr) {
		t.Fatal("nothing may be written")
	}
	for _, definition := range provider.requests[0].Tools {
		for keyword := range definition.Parameters {
			switch keyword {
			case "type", "properties", "required", "additionalProperties", "description", "items", "enum":
			default:
				t.Errorf("%s sends unsupported keyword %q", definition.Name, keyword)
			}
		}
	}
}

func TestModelChosenCallIDsAreReplaced(t *testing.T) {
	envelope := func(path string) llm.CompletionResponse {
		return llm.CompletionResponse{Content: `<boi-action>{"id":"unique-id","tool":"workspace.read","purpose":"read","arguments":{"path":"` + path + `"}}</boi-action>`}
	}
	provider := &scriptedNativeProvider{responses: []llm.CompletionResponse{envelope("a.txt"), envelope("b.txt"), {Content: "both read"}}}
	service, root := nativeService(t, provider)
	for name, body := range map[string]string{"a.txt": "alpha", "b.txt": "bravo"} {
		if err := os.WriteFile(filepath.Join(root, name), []byte(body), 0o600); err != nil {
			t.Fatal(err)
		}
	}
	result, err := service.Run(context.Background(), "read both")
	if err != nil {
		t.Fatal(err)
	}
	seen := map[string]bool{}
	for _, step := range result.Trace {
		if step.ToolCall == nil {
			continue
		}
		if step.ToolCall.ID == "unique-id" || seen[step.ToolCall.ID] {
			t.Fatalf("a repeated model ID reached the Broker: %q", step.ToolCall.ID)
		}
		seen[step.ToolCall.ID] = true
	}
	last := provider.requests[2].Messages
	transcript := ""
	for _, message := range last {
		transcript += message.Content
	}
	if !strings.Contains(transcript, "alpha") || !strings.Contains(transcript, "bravo") {
		t.Fatal("each call must keep its own observation")
	}
}

func approveAll(t *testing.T, service *Service, query string) *AgentResult {
	t.Helper()
	var result *AgentResult
	for event := range service.Start(context.Background(), query) {
		if event.Approval != nil {
			event.Approval.Decisions <- ApprovalDecision{RequestID: event.Approval.Request.ID, State: ApprovalApproved, DecidedAt: time.Now()}
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

func TestNativeToolCallingChallengesACodeFenceAnswerAndRecordsTheRealChange(t *testing.T) {
	provider := &scriptedNativeProvider{responses: []llm.CompletionResponse{
		{ToolCalls: []llm.ToolCallRequest{{ID: "c0", Name: "workspace__read", Arguments: `{"path":"calc.go"}`}}},
		{Content: "Change it like this:\n```go\nreturn a + b\n```"},
		{ToolCalls: []llm.ToolCallRequest{{ID: "c1", Name: "workspace__edit", Arguments: `{"path":"calc.go","old":"a - b","new":"a + b"}`}}},
		{Content: "Applied the fix to calc.go."},
	}}
	service, root := nativeService(t, provider)
	if err := os.WriteFile(filepath.Join(root, "calc.go"), []byte("package bench\n\nfunc Add(a, b int) int { return a - b }\n"), 0o600); err != nil {
		t.Fatal(err)
	}
	result := approveAll(t, service, "fix the bug in calc.go")
	if data, _ := os.ReadFile(filepath.Join(root, "calc.go")); !strings.Contains(string(data), "a + b") {
		t.Fatalf("the fix was not applied: %q", data)
	}
	if result.Outcome != OutcomeChanged || len(result.ChangedPaths) != 1 || result.ChangedPaths[0] != "calc.go" || result.Notice != "" {
		t.Fatalf("result = %+v", result)
	}
	if len(provider.requests) != 4 {
		t.Fatalf("provider calls = %d, want 4 (read, fence challenged, edit, answer)", len(provider.requests))
	}
	last := provider.requests[2].Messages
	if !strings.Contains(last[len(last)-1].Content, "your reply came with no change") {
		t.Fatalf("the challenge did not reach the model in native mode: %q", last[len(last)-1].Content)
	}
}
