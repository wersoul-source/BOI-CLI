package agent

import (
	"context"
	"fmt"
	"os"
	"path/filepath"
	"strings"
	"testing"
	"time"

	"github.com/boi-family/boi-cli/internal/equipment/tools/mcp"
)

func TestParseDecisionRejectsModelSecurityFields(t *testing.T) {
	_, err := ParseDecision(`<boi-action>{"id":"1","tool":"workspace.read","purpose":"inspect","arguments":{"path":"a"},"risk":"read"}</boi-action>`)
	if err == nil {
		t.Fatal("expected model-controlled risk field to be rejected")
	}
}

func TestParseDecisionAcceptsStrictSkillEnvelope(t *testing.T) {
	decision, err := ParseDecision(`<boi-skill>{"name":"git-helper"}</boi-skill>`)
	if err != nil || decision.Kind != DecisionUseSkill || decision.SkillName != "git-helper" {
		t.Fatalf("decision=%#v err=%v", decision, err)
	}
	if _, err := ParseDecision(`text <boi-skill>{"name":"git-helper"}</boi-skill>`); err == nil {
		t.Fatal("accepted mixed Skill envelope")
	}
	if _, err := ParseDecision(`<boi-skill>{"name":"git-helper"}</boi-skill><boi-action>{}</boi-action>`); err == nil {
		t.Fatal("accepted Skill and Tool together")
	}
}

func TestBrokerCapabilityProfileCanDisableAllTools(t *testing.T) {
	b := testBroker(t)
	b.SetToolCallingAllowed(false)
	if len(b.CapabilityNames()) != 0 {
		t.Fatal("disabled Tool Calling must expose no capability names")
	}
	if _, err := b.Prepare(ToolCall{ID: "1", Tool: "workspace.read", Purpose: "read", Arguments: map[string]any{"path": "README.md"}}); err == nil {
		t.Fatal("disabled Tool Calling must reject preparation")
	}
}

func TestBrokerRejectsSixteenthActiveTool(t *testing.T) {
	b := NewBroker()
	names := make([]string, 16)
	for index := range names {
		names[index] = fmt.Sprintf("tool-%02d", index)
		if err := b.Register(&fakeTool{name: names[index]}); err != nil {
			t.Fatal(err)
		}
	}
	if err := b.SetActiveCapabilities(names); err == nil {
		t.Fatal("expected active Tool limit rejection")
	}
}

func TestParseDecisionRejectsTrailingJSON(t *testing.T) {
	_, err := ParseDecision(`<boi-action>{"id":"1","tool":"workspace.read","purpose":"inspect","arguments":{"path":"a"}} {"extra":true}</boi-action>`)
	if err == nil {
		t.Fatal("expected trailing JSON to be rejected")
	}
}

func TestBrokerAssignsHostRiskAndApproval(t *testing.T) {
	b := testBroker(t)
	call, err := b.Prepare(ToolCall{ID: "1", Tool: "workspace.write", Purpose: "save", Arguments: map[string]any{"path": "note.txt", "content": "hello"}, Risk: RiskRead, Approval: ApprovalAuto})
	if err != nil {
		t.Fatal(err)
	}
	if call.Risk != RiskChange || call.Approval != ApprovalConfirm || call.Timeout <= 0 {
		t.Fatalf("host policy not applied: %#v", call)
	}
}

func TestBrokerPreservesHostIdempotencyNamespace(t *testing.T) {
	b := testBroker(t)
	call, err := b.Prepare(ToolCall{ID: "write-1", Tool: "workspace.write", Purpose: "save", Arguments: map[string]any{"path": "note.txt", "content": "hello"}, IdempotencyKey: "automation:host-owned"})
	if err != nil {
		t.Fatal(err)
	}
	if call.IdempotencyKey != "automation:host-owned" {
		t.Fatalf("idempotency key=%q", call.IdempotencyKey)
	}
}

func TestBrokerDisablesLocalCapabilitiesWithoutWorkspace(t *testing.T) {
	b := NewBroker()
	if _, err := b.Prepare(ToolCall{ID: "1", Tool: "process.run", Purpose: "run", Arguments: map[string]any{"command": "echo unsafe"}}); err == nil {
		t.Fatal("process capability enabled without a registered Tool")
	}
}

func TestBrokerBlocksMutationAfterApproval(t *testing.T) {
	b := testBroker(t)
	call, err := b.Prepare(ToolCall{ID: "1", Tool: "workspace.write", Purpose: "save", Arguments: map[string]any{"path": "note.txt", "content": "hello"}})
	if err != nil {
		t.Fatal(err)
	}
	request, err := NewApprovalRequest("approval_1", call, time.Now(), time.Now().Add(time.Minute))
	if err != nil {
		t.Fatal(err)
	}
	call.Arguments["content"] = "mutated"
	_, err = b.Act(context.Background(), call, Authorization{Allowed: true, State: ApprovalApproved, Request: &request})
	if err == nil || (!strings.Contains(err.Error(), "fingerprint") && !strings.Contains(err.Error(), "host-authorized")) {
		t.Fatalf("expected fingerprint error, got %v", err)
	}
}

func TestBrokerWritesOnlyAfterExactApproval(t *testing.T) {
	b := testBroker(t)
	call, err := b.Prepare(ToolCall{ID: "1", Tool: "workspace.write", Purpose: "save", Arguments: map[string]any{"path": "note.txt", "content": "hello"}})
	if err != nil {
		t.Fatal(err)
	}
	request, err := NewApprovalRequest("approval_1", call, time.Now(), time.Now().Add(time.Minute))
	if err != nil {
		t.Fatal(err)
	}
	if _, err := b.Act(context.Background(), call, Authorization{Allowed: false}); err == nil {
		t.Fatal("write executed without approval")
	}
	result, err := b.Act(context.Background(), call, Authorization{Allowed: true, State: ApprovalApproved, Request: &request})
	if err != nil || result.Status != ToolSucceeded {
		t.Fatalf("result=%#v err=%v", result, err)
	}
	content, err := os.ReadFile(filepath.Join(b.root, "note.txt"))
	if err != nil || string(content) != "hello" {
		t.Fatalf("content=%q err=%v", content, err)
	}
}

func TestInteractiveAuthorizerWaitsForMatchingDecision(t *testing.T) {
	call, err := testBroker(t).Prepare(ToolCall{ID: "1", Tool: "workspace.write", Purpose: "save", Arguments: map[string]any{"path": "note.txt", "content": "hello"}})
	if err != nil {
		t.Fatal(err)
	}
	authorizer := &InteractiveAuthorizer{TTL: time.Second, Emit: func(event ApprovalEvent) error {
		event.Decisions <- ApprovalDecision{RequestID: event.Request.ID, State: ApprovalApproved, DecidedAt: time.Now()}
		return nil
	}}
	auth, err := authorizer.Authorize(context.Background(), call)
	if err != nil || !auth.Allowed || auth.Request == nil {
		t.Fatalf("auth=%#v err=%v", auth, err)
	}
}

type fakeExternalInvoker struct{ calls int }

func (f *fakeExternalInvoker) CallTool(context.Context, string, string, map[string]any) (string, error) {
	f.calls++
	return `{"content":"ok"}`, nil
}

func TestMCPToolIsExternalAndApprovalGated(t *testing.T) {
	b := testBroker(t)
	invoker := &fakeExternalInvoker{}
	tools, err := mcp.Tools("docs", []string{"search"}, invoker)
	if err != nil {
		t.Fatal(err)
	}
	for _, tool := range tools {
		if err := b.Register(tool); err != nil {
			t.Fatal(err)
		}
	}
	if _, err := b.Prepare(ToolCall{ID: "m0", Tool: "mcp.docs.search", Purpose: "probe", Arguments: map[string]any{"query": "agent"}}); err == nil {
		t.Fatal("registered MCP Tool became active without registry selection")
	}
	if err := b.SetActiveCapabilities([]string{"mcp.docs.search"}); err != nil {
		t.Fatal(err)
	}
	call, err := b.Prepare(ToolCall{ID: "m1", Tool: "mcp.docs.search", Purpose: "look up documentation", Arguments: map[string]any{"query": "agent"}})
	if err != nil {
		t.Fatal(err)
	}
	if call.Risk != RiskExternal || call.Approval != ApprovalConfirm {
		t.Fatalf("unsafe MCP policy: %#v", call)
	}
	if _, err := b.Act(context.Background(), call, Authorization{Allowed: false}); err == nil {
		t.Fatal("MCP invoked without approval")
	}
	if invoker.calls != 0 {
		t.Fatal("MCP invoker called before approval")
	}
	request, err := NewApprovalRequest("approval_m1", call, time.Now(), time.Now().Add(time.Minute))
	if err != nil {
		t.Fatal(err)
	}
	result, err := b.Act(context.Background(), call, Authorization{Allowed: true, State: ApprovalApproved, Request: &request})
	if err != nil || result.Status != ToolSucceeded || invoker.calls != 1 {
		t.Fatalf("result=%#v calls=%d err=%v", result, invoker.calls, err)
	}
}

func TestBrokerRejectsDuplicateAndInvalidTools(t *testing.T) {
	b := NewBroker()
	if err := b.Register(&fakeTool{name: "x"}); err != nil {
		t.Fatal(err)
	}
	if err := b.Register(&fakeTool{name: "x"}); err == nil {
		t.Fatal("duplicate Tool name accepted")
	}
	if err := b.Register(&fakeTool{name: ""}); err == nil {
		t.Fatal("unnamed Tool accepted")
	}
	if err := b.Register(nil); err == nil {
		t.Fatal("nil Tool accepted")
	}
}

func TestNewToolNeedsNoBrokerChange(t *testing.T) {
	b := NewBroker()
	tool := &fakeTool{name: "custom.echo"}
	if err := b.Register(tool); err != nil {
		t.Fatal(err)
	}
	if err := b.SetActiveCapabilities([]string{"custom.echo"}); err != nil {
		t.Fatal(err)
	}
	call, err := b.Prepare(ToolCall{ID: "c1", Tool: "custom.echo", Purpose: "probe"})
	if err != nil {
		t.Fatal(err)
	}
	result, err := b.Act(context.Background(), call, Authorization{})
	if err != nil || result.Output != "ok" || tool.calls != 1 {
		t.Fatalf("result=%#v err=%v", result, err)
	}
	if !strings.Contains(ToolPrompt(b), "custom.echo()") {
		t.Fatal("ToolPrompt must list the registered Tool's usage")
	}
}
