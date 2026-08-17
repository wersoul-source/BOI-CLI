package tui

import (
	"context"
	"testing"
)

func TestHandleAgentResponseCancellationReturnsToIdle(t *testing.T) {
	cancelCalled := false
	m := &Model{
		chat:   NewChat(),
		status: NewStatus("kamkaew", "test", []string{"kamkaew"}),
		cancelActive: func() {
			cancelCalled = true
		},
	}
	m.status.SetStatus("cancelling")

	m.handleAgentResponse(agentResponseMsg{err: context.Canceled})

	if !cancelCalled {
		t.Fatal("expected active context to be released")
	}
	if m.cancelActive != nil {
		t.Fatal("expected active cancellation function to be cleared")
	}
	if got := m.status.Status(); got != "idle" {
		t.Fatalf("status = %q, want idle", got)
	}
	if len(m.chat.messages) != 1 || m.chat.messages[0].Content != "Agent task cancelled." {
		t.Fatalf("unexpected cancellation message: %#v", m.chat.messages)
	}
}

func TestBusyIncludesWorkspaceAndCancellationStates(t *testing.T) {
	m := &Model{status: NewStatus("kamkaew", "test", []string{"kamkaew"})}

	for _, status := range []string{"thinking", "working", "cancelling"} {
		m.status.SetStatus(status)
		if !m.isBusy() {
			t.Fatalf("status %q should be busy", status)
		}
	}

	m.status.SetStatus("idle")
	if m.isBusy() {
		t.Fatal("idle status should not be busy")
	}
}
