package agent

import (
	"context"
	"os"
	"path/filepath"
	"strings"
	"testing"
	"time"

	"github.com/boi-family/boi-cli/internal/workspace"
)

func testBroker(t *testing.T) *Broker {
	t.Helper()
	sandbox, err := workspace.NewSandbox(t.TempDir())
	if err != nil {
		t.Fatal(err)
	}
	return NewBroker(sandbox)
}

func TestParseDecisionRejectsModelSecurityFields(t *testing.T) {
	_, err := ParseDecision(`<boi-action>{"id":"1","tool":"workspace.read","purpose":"inspect","arguments":{"path":"a"},"risk":"read"}</boi-action>`)
	if err == nil {
		t.Fatal("expected model-controlled risk field to be rejected")
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

func TestBrokerDisablesLocalCapabilitiesWithoutWorkspace(t *testing.T) {
	b := NewBroker(nil)
	if _, err := b.Prepare(ToolCall{ID: "1", Tool: "process.run", Purpose: "run", Arguments: map[string]any{"command": "echo unsafe"}}); err == nil {
		t.Fatal("process capability enabled without workspace")
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
	content, err := os.ReadFile(filepath.Join(b.sandbox.Root(), "note.txt"))
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
	if err := b.RegisterMCP("docs", []string{"search"}, invoker); err != nil {
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
