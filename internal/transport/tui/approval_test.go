package tui

import (
	"strings"
	"testing"
	"time"

	"github.com/boi-family/boi-cli/internal/agent"
	tea "github.com/charmbracelet/bubbletea"
)

func approvalFixture(now time.Time) agent.ApprovalRequest {
	call := agent.ToolCall{
		ID:             "call_001",
		Tool:           "filesystem.write",
		Purpose:        "update the Agent timeout",
		Arguments:      map[string]any{"path": "internal/agent/service.go"},
		Target:         "internal/agent/service.go",
		ExpectedResult: "the timeout is updated",
		Preview:        "- timeout: 30s\n+ timeout: 120s",
		Risk:           agent.RiskChange,
		Approval:       agent.ApprovalConfirm,
		Timeout:        10 * time.Second,
	}
	request, err := agent.NewApprovalRequest(
		"approval_001",
		call,
		now,
		now.Add(time.Minute),
	)
	if err != nil {
		panic(err)
	}
	return request
}

func TestApprovalModelShowsExactActionAndRequiresExplicitKey(t *testing.T) {
	now := time.Now()
	model := NewApproval()
	model.SetWidth(80)
	if err := model.Open(approvalFixture(now), now); err != nil {
		t.Fatalf("open approval: %v", err)
	}

	view := model.View()
	for _, want := range []string{
		"APPROVAL REQUIRED",
		"filesystem.write",
		"internal/agent/service.go",
		"Risk: CHANGE",
		"Approve once",
	} {
		if !strings.Contains(view, want) {
			t.Fatalf("approval view does not contain %q:\n%s", want, view)
		}
	}

	if decision, handled := model.Decide("enter", now); handled || decision != nil {
		t.Fatal("enter must not approve a request")
	}
	decision, handled := model.Decide("a", now)
	if !handled || decision.State != agent.ApprovalApproved {
		t.Fatalf("unexpected approve decision: %#v, handled=%v", decision, handled)
	}
}

func TestApprovalModelRejectAndCancel(t *testing.T) {
	now := time.Now()
	for _, tc := range []struct {
		key  string
		want agent.ApprovalState
	}{
		{"r", agent.ApprovalRejected},
		{"esc", agent.ApprovalCancelled},
	} {
		t.Run(tc.key, func(t *testing.T) {
			model := NewApproval()
			if err := model.Open(approvalFixture(now), now); err != nil {
				t.Fatalf("open approval: %v", err)
			}
			decision, handled := model.Decide(tc.key, now)
			if !handled || decision.State != tc.want {
				t.Fatalf("decision = %#v, handled=%v, want %s", decision, handled, tc.want)
			}
		})
	}
}

func TestApprovalModelExpiresBeforeDecision(t *testing.T) {
	now := time.Now()
	model := NewApproval()
	request := approvalFixture(now)
	if err := model.Open(request, now); err != nil {
		t.Fatalf("open approval: %v", err)
	}
	decision, handled := model.Decide("a", request.ExpiresAt)
	if !handled || decision.State != agent.ApprovalExpired {
		t.Fatalf("decision = %#v, handled=%v, want expired", decision, handled)
	}
}

func TestMainModelApprovalLifecycle(t *testing.T) {
	now := time.Now()
	decisions := make(chan agent.ApprovalDecision, 1)
	m := &Model{
		chat:     NewChat(),
		input:    NewInput(),
		status:   NewStatus("kamkaew", "test", []string{"kamkaew"}),
		help:     NewHelp(),
		approval: NewApproval(),
		mode:     "chat",
		width:    80,
		height:   30,
	}

	_, cmd := m.Update(approvalRequestedMsg{
		request:   approvalFixture(now),
		decisions: decisions,
	})
	if cmd != nil {
		t.Fatal("opening approval should not perform I/O")
	}
	if !m.approval.Active() || m.status.Status() != "approval" {
		t.Fatalf("approval was not activated: active=%v status=%s", m.approval.Active(), m.status.Status())
	}

	_, cmd = m.Update(tea.KeyMsg{Type: tea.KeyRunes, Runes: []rune{'a'}})
	if cmd == nil {
		t.Fatal("approval decision should return a delivery command")
	}
	msg := cmd()
	if _, ok := msg.(approvalDecisionSentMsg); !ok {
		t.Fatalf("unexpected decision message: %T", msg)
	}
	m.Update(msg)

	select {
	case decision := <-decisions:
		if decision.State != agent.ApprovalApproved {
			t.Fatalf("decision state = %s, want approved", decision.State)
		}
	default:
		t.Fatal("approval decision was not delivered")
	}
	if m.approval.Active() {
		t.Fatal("approval panel should be closed after a decision")
	}
	if m.status.Status() != "thinking" {
		t.Fatalf("status = %s, want thinking while Agent resumes", m.status.Status())
	}
}

func TestMainModelApprovalExpiresOnTick(t *testing.T) {
	now := time.Now()
	request := approvalFixture(now)
	decisions := make(chan agent.ApprovalDecision, 1)
	m := &Model{
		chat:     NewChat(),
		input:    NewInput(),
		status:   NewStatus("kamkaew", "test", []string{"kamkaew"}),
		help:     NewHelp(),
		approval: NewApproval(),
		mode:     "chat",
		width:    80,
		height:   30,
	}
	m.Update(approvalRequestedMsg{request: request, decisions: decisions})

	_, cmd := m.Update(tickMsg(request.ExpiresAt))
	if cmd == nil {
		t.Fatal("expiration should schedule decision delivery and the next tick")
	}
	if m.approval.Active() {
		t.Fatal("expired approval should close without keyboard input")
	}
}
