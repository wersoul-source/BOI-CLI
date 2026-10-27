package agent

import (
	"context"
	"os"
	"path/filepath"
	"strconv"
	"strings"
	"testing"
	"time"
	"unicode/utf8"

	"github.com/boi-family/boi-cli/internal/core/persona"
	llm "github.com/boi-family/boi-cli/internal/runtime/llm"
	"github.com/boi-family/boi-cli/internal/runtime/workspace"
)

type captureProvider struct {
	request llm.CompletionRequest
}

type captureTaskRecorder struct {
	session   *TaskSession
	events    []EngineEvent
	finalized *AgentResult
}

func (r *captureTaskRecorder) Begin(_ string, _ *TaskPlan) (*TaskSession, error) {
	return r.session, nil
}

func (r *captureTaskRecorder) RecordEvent(_ *TaskSession, event EngineEvent) error {
	r.events = append(r.events, event)
	return nil
}

func (r *captureTaskRecorder) Finalize(_ *TaskSession, result *AgentResult) error {
	r.finalized = result
	result.Manifest = "agent-folder/output/task-test/manifest.json"
	return nil
}

func (p *captureProvider) Name() string { return "capture" }

func (p *captureProvider) Complete(_ context.Context, req llm.CompletionRequest) (*llm.CompletionResponse, error) {
	p.request = req
	return &llm.CompletionResponse{
		Content:      "ok",
		InputTokens:  2,
		OutputTokens: 3,
		Provider:     "capture",
		Model:        "test-model",
	}, nil
}

func (p *captureProvider) Stream(context.Context, llm.CompletionRequest) (<-chan llm.Token, error) {
	return nil, nil
}

func TestServiceRunsInsideWorkspaceContext(t *testing.T) {
	t.Parallel()

	root := t.TempDir()
	if _, err := os.Stat(root); err != nil {
		t.Fatalf("stat workspace: %v", err)
	}
	sandbox, err := workspace.NewSandbox(root)
	if err != nil {
		t.Fatalf("create sandbox: %v", err)
	}
	provider := &captureProvider{}
	service := NewService(
		&persona.Persona{Name: "test", SystemPrompt: "persona prompt", MaxTokens: 123, Temperature: 0.2},
		llm.NewRouter([]llm.Provider{provider}),
		nil,
		sandbox,
	)

	result, err := service.Run(context.Background(), "hello")
	if err != nil {
		t.Fatalf("run service: %v", err)
	}
	if result.Response != "ok" || result.Provider != "capture" || result.Model != "test-model" || result.Tokens != 5 {
		t.Fatalf("unexpected result: %#v", result)
	}
	if len(provider.request.Messages) != 2 {
		t.Fatalf("messages = %#v", provider.request.Messages)
	}
	systemPrompt := provider.request.Messages[0].Content
	if !strings.Contains(systemPrompt, "persona prompt") || !strings.Contains(systemPrompt, strconv.Quote(sandbox.Root())) {
		t.Fatalf("system prompt does not contain persona and workspace boundary: %q", systemPrompt)
	}
	if provider.request.MaxTokens != 123 || provider.request.Temperature != 0.2 {
		t.Fatalf("persona settings not applied: %#v", provider.request)
	}
}

func TestServiceRequiresProvider(t *testing.T) {
	t.Parallel()

	service := NewService(persona.DefaultPersona(), nil, nil, nil)
	if _, err := service.Run(context.Background(), "hello"); err != ErrNoProvider {
		t.Fatalf("Run() error = %v, want ErrNoProvider", err)
	}
}

func TestServiceComposesAgentFolderScopeAndProviderProfile(t *testing.T) {
	root := t.TempDir()
	provider := &captureProvider{}
	sandbox, err := workspace.NewSandbox(root)
	if err != nil {
		t.Fatal(err)
	}
	recorder := &captureTaskRecorder{session: &TaskSession{
		ID: "task-test", BinDir: filepath.Join(root, "agent-folder", "bin", "task-test"),
		OutputDir: filepath.Join(root, "agent-folder", "output", "task-test"), StartedAt: time.Now(),
	}}
	service := withWorkspaceTools(t, NewService(persona.DefaultPersona(), llm.NewRouter([]llm.Provider{provider}), nil, sandbox), sandbox)
	service.SetTaskRecorder(recorder)
	service.SetProviderProfileReference("capture", "test-model", ".boi/provider-profiles/test.json")
	result, err := service.Run(context.Background(), "produce a report")
	if err != nil {
		t.Fatal(err)
	}
	if result.TaskID != "task-test" || result.Manifest == "" || result.ProviderProfileRef != ".boi/provider-profiles/test.json" || recorder.finalized != result {
		t.Fatalf("Agent Folder result = %#v", result)
	}
	if len(recorder.events) == 0 {
		t.Fatal("runtime events were not checkpointed")
	}
	prompt := provider.request.Messages[0].Content
	if !strings.Contains(prompt, "agent-folder/bin/task-test") || !strings.Contains(prompt, "agent-folder/output/task-test") || !strings.Contains(prompt, "additional filesystem authority") {
		t.Fatalf("Agent Folder scope missing from prompt: %q", prompt)
	}
}

func TestAutomationMutationIsDeniedWithoutWaitingOrWriting(t *testing.T) {
	root := t.TempDir()
	sandbox, err := workspace.NewSandbox(root)
	if err != nil {
		t.Fatal(err)
	}
	provider := &sequenceProvider{responses: []string{`<boi-action>{"id":"write-automation","tool":"workspace.write","purpose":"save","arguments":{"path":"blocked.txt","content":"no"}}</boi-action>`}}
	service := withWorkspaceTools(t, NewService(persona.DefaultPersona(), llm.NewRouter([]llm.Provider{provider}), nil, sandbox), sandbox)
	result, err := service.RunAutomation(context.Background(), "write a file", "automation-001")
	if err == nil || result == nil || result.StopReason != StopNeedsApproval {
		t.Fatalf("result=%#v err=%v", result, err)
	}
	if result.IdempotencyKeyHash != automationKeyHash("automation-001") {
		t.Fatalf("idempotency hash=%q", result.IdempotencyKeyHash)
	}
	if _, statErr := os.Stat(filepath.Join(root, "blocked.txt")); !os.IsNotExist(statErr) {
		t.Fatalf("non-interactive mutation executed: %v", statErr)
	}
}

func TestServiceInteractiveApprovalCompletesWriteLoop(t *testing.T) {
	root := t.TempDir()
	sandbox, err := workspace.NewSandbox(root)
	if err != nil {
		t.Fatal(err)
	}
	provider := &sequenceProvider{responses: []string{`<boi-action>{"id":"write-1","tool":"workspace.write","purpose":"save note","arguments":{"path":"note.txt","content":"hello"}}</boi-action>`, "saved"}}
	service := withWorkspaceTools(t, NewService(persona.DefaultPersona(), llm.NewRouter([]llm.Provider{provider}), nil, sandbox), sandbox)
	ctx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
	defer cancel()
	events := service.Start(ctx, "save note")
	var approval RuntimeEvent
	kinds := map[string]bool{}
	for event := range events {
		if event.Engine != nil {
			kinds[event.Engine.Kind] = true
		}
		if event.Approval != nil {
			kinds["approval_request"] = true
			approval = event
			break
		}
	}
	if approval.Approval == nil {
		t.Fatalf("expected approval, got %#v", approval)
	}
	approval.Approval.Decisions <- ApprovalDecision{RequestID: approval.Approval.Request.ID, State: ApprovalApproved, DecidedAt: time.Now()}
	var completed RuntimeEvent
	for event := range events {
		if event.Engine != nil {
			kinds[event.Engine.Kind] = true
		}
		if event.Result != nil || event.Err != nil {
			completed = event
			break
		}
	}
	if completed.Err != nil || completed.Result == nil || completed.Result.Response != "saved" {
		t.Fatalf("unexpected completion: %#v", completed)
	}
	content, err := os.ReadFile(filepath.Join(root, "note.txt"))
	if err != nil || string(content) != "hello" {
		t.Fatalf("content=%q err=%v", content, err)
	}
	for _, kind := range []string{"task", "tool", "approval_request", "approval", "tool_result", "verification", "stop"} {
		if !kinds[kind] {
			t.Fatalf("missing runtime event %q: %#v", kind, kinds)
		}
	}
}

func newSequenceService(t *testing.T, responses ...string) (*Service, *sequenceProvider, string) {
	t.Helper()
	root := t.TempDir()
	sandbox, err := workspace.NewSandbox(root)
	if err != nil {
		t.Fatal(err)
	}
	provider := &sequenceProvider{responses: responses}
	return withWorkspaceTools(t, NewService(persona.DefaultPersona(), llm.NewRouter([]llm.Provider{provider}), nil, sandbox), sandbox), provider, root
}

func allContent(request llm.CompletionRequest) string {
	var parts []string
	for _, message := range request.Messages {
		parts = append(parts, message.Content)
	}
	return strings.Join(parts, "\n")
}

func TestMultiStepTaskKeepsEveryObservation(t *testing.T) {
	service, provider, root := newSequenceService(t,
		`<boi-action>{"id":"r1","tool":"workspace.read","purpose":"read a","arguments":{"path":"a.txt"}}</boi-action>`,
		`<boi-action>{"id":"r2","tool":"workspace.read","purpose":"read b","arguments":{"path":"b.txt"}}</boi-action>`,
		"both read",
	)
	_ = os.WriteFile(filepath.Join(root, "a.txt"), []byte("ALPHA-1"), 0o600)
	_ = os.WriteFile(filepath.Join(root, "b.txt"), []byte("BRAVO-2"), 0o600)
	if _, err := service.Run(context.Background(), "compare a and b"); err != nil {
		t.Fatal(err)
	}
	last := allContent(provider.requests[2])
	if !strings.Contains(last, "ALPHA-1") || !strings.Contains(last, "BRAVO-2") {
		t.Fatal("third decision lost the first file's content")
	}
}

func TestRejectedReplyIsExplainedAndCounted(t *testing.T) {
	service, provider, _ := newSequenceService(t,
		`<boi-action>{"id":"1","tool":"workspace.read","purpose":"x","risk":"read"}</boi-action>`,
		"fixed answer",
	)
	result, err := service.Run(context.Background(), "hello")
	if err != nil || result.Response != "fixed answer" {
		t.Fatalf("result=%#v err=%v", result, err)
	}
	if !strings.Contains(allContent(provider.requests[1]), "previous reply was rejected") {
		t.Fatal("retry did not tell the model why its reply was rejected")
	}
	if result.Usage.ProviderCalls != 2 {
		t.Fatalf("provider calls = %d, want 2", result.Usage.ProviderCalls)
	}
}

func TestInteractiveConversationIsRememberedAndResettable(t *testing.T) {
	service, provider, _ := newSequenceService(t, "my name is Kite", "you asked my name", "fresh")
	runStart := func(query string) {
		for event := range service.Start(context.Background(), query) {
			if event.Err != nil {
				t.Fatal(event.Err)
			}
		}
	}
	runStart("what is your name?")
	runStart("what did I just ask?")
	second := allContent(provider.requests[1])
	if !strings.Contains(second, "what is your name?") || !strings.Contains(second, "my name is Kite") {
		t.Fatal("follow-up turn lost the previous turn")
	}
	service.ResetConversation()
	runStart("new topic")
	if strings.Contains(allContent(provider.requests[2]), "my name is Kite") {
		t.Fatal("ResetConversation did not forget prior turns")
	}
}

func TestAutomationRunsDoNotShareConversation(t *testing.T) {
	service, provider, _ := newSequenceService(t, "ANSWER-ZULU-7", "ANSWER-YANKEE-8")
	_, _ = service.Run(context.Background(), "one")
	_, _ = service.Run(context.Background(), "two")
	if strings.Contains(allContent(provider.requests[1]), "ANSWER-ZULU-7") {
		t.Fatal("non-interactive run leaked a previous run's answer")
	}
}

func TestClipMiddleKeepsTailAndValidUTF8(t *testing.T) {
	text := strings.Repeat("ไทย", 4000) + `"Truncated":true}`
	clipped := clipMiddle(text, 1000)
	if !strings.HasSuffix(clipped, `"Truncated":true}`) || !utf8.ValidString(clipped) || len(clipped) > 1100 {
		t.Fatalf("bad clip: len=%d valid=%v", len(clipped), utf8.ValidString(clipped))
	}
}

func startResult(t *testing.T, service *Service, query string) *AgentResult {
	t.Helper()
	var result *AgentResult
	for event := range service.Start(context.Background(), query) {
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

func TestAnswerAboutAFileWithoutAnyToolIsChallengedAndFlagged(t *testing.T) {
	service, provider, root := newSequenceService(t, "I fixed calc.go", "Yes, calc.go is fixed")
	_ = os.WriteFile(filepath.Join(root, "calc.go"), []byte("package bench\n"), 0o600)
	result := startResult(t, service, "fix the bug in calc.go")
	if result.Usage.ProviderCalls != 2 || len(provider.requests) != 2 {
		t.Fatalf("the model was not asked again: %d provider calls", result.Usage.ProviderCalls)
	}
	if !strings.Contains(allContent(provider.requests[1]), "your reply came with no change") {
		t.Fatal("the second request does not carry the host's challenge")
	}
	if result.StopReason != StopCompleted || result.Notice == "" || result.Response != "Yes, calc.go is fixed" {
		t.Fatalf("result = %#v", result)
	}
}

func TestPlainConversationIsNotChallenged(t *testing.T) {
	service, provider, root := newSequenceService(t, "my name is Kite")
	_ = os.WriteFile(filepath.Join(root, "calc.go"), []byte("package bench\n"), 0o600)
	result := startResult(t, service, "what is your name?")
	if result.Usage.ProviderCalls != 1 || len(provider.requests) != 1 || result.Notice != "" {
		t.Fatalf("a chat turn was challenged: calls=%d notice=%q", result.Usage.ProviderCalls, result.Notice)
	}
}

// A non-interactive run denies every mutation Tool, so asking the model again
// cannot change anything and the challenge is skipped. The host still records
// that nothing was changed or run, because an automation run that quietly
// reports success it never earned is the worst place to stay silent.
func TestNonInteractiveRunsAreNotChallengedButAreStillRecorded(t *testing.T) {
	service, _, root := newSequenceService(t, "I fixed calc.go")
	_ = os.WriteFile(filepath.Join(root, "calc.go"), []byte("package bench\n"), 0o600)
	result, err := service.Run(context.Background(), "fix the bug in calc.go")
	if err != nil || result.Usage.ProviderCalls != 1 {
		t.Fatalf("a non-interactive run was challenged: %#v %v", result, err)
	}
	if result.Notice == "" || result.Outcome != OutcomeBlocked {
		t.Fatalf("an unearned success was recorded as clean: outcome=%q notice=%q", result.Outcome, result.Notice)
	}
}

// The reported failure: a Provider that is not qualified for Tool calling has
// every Tool withheld, so CanChange was false and the grounding check switched
// itself off in exactly the case it exists for. The run then finished as a
// clean completion while nothing had been done.
func TestWithheldToolsAreReportedBlockedNotCompleted(t *testing.T) {
	service, provider, root := newSequenceService(t, "Done. I created report.json for you.")
	_ = os.WriteFile(filepath.Join(root, "calc.go"), []byte("package bench\n"), 0o600)
	service.SetToolCallingAllowed(false)
	result := startResult(t, service, "write report.json describing calc.go")
	if result.Outcome != OutcomeBlocked {
		t.Fatalf("outcome = %q, want blocked", result.Outcome)
	}
	if !strings.Contains(result.Notice, "not qualified for Tool calling") {
		t.Fatalf("notice should name the Provider as the cause, got %q", result.Notice)
	}
	if len(provider.requests) != 1 {
		t.Fatalf("challenging a model that has no Tool is wasted: %d calls", len(provider.requests))
	}
}

// A task that asks for a file to be produced names nothing that exists yet, so
// the old check could never see it and "I created it" was never questioned.
func TestATaskToProduceAFileIsChallenged(t *testing.T) {
	service, provider, _ := newSequenceService(t, "I created report.json", "I could not create it")
	result := startResult(t, service, "create report.json in the workspace")
	if len(provider.requests) != 2 {
		t.Fatalf("an unproduced file was not challenged: %d calls", len(provider.requests))
	}
	if result.Outcome != OutcomeBlocked || result.Notice == "" {
		t.Fatalf("result = %q / %q", result.Outcome, result.Notice)
	}
}

// The target shape must not fire on prose. A version and a web address are not
// files a task asks for.
func TestFileTargetShapeIgnoresVersionsAndURLs(t *testing.T) {
	for _, token := range []string{"2.6.1", "v2.6.1", "https://example.com/docs/api", "http://a.b/c", "ok", "1.2"} {
		if looksLikeFileTarget(token) {
			t.Errorf("%q should not read as a file target", token)
		}
	}
	for _, token := range []string{"report.json", "OK.docx", "system_probe.py", "agent-trial-001/probe.py", "docs/guide.md"} {
		if !looksLikeFileTarget(token) {
			t.Errorf("%q should read as a file target", token)
		}
	}
}
