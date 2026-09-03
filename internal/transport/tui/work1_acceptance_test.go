package tui

import (
	"strings"
	"testing"

	"github.com/boi-family/boi-cli/internal/agent"
	tea "github.com/charmbracelet/bubbletea"
)

func TestWork1HeadlessRenderAndResultFlow(t *testing.T) {
	m := &Model{
		chat:     NewChat(),
		input:    NewInput(),
		status:   NewStatus("บ๋อยทดสอบ", "fixture/model", nil),
		help:     NewHelp(),
		approval: NewApproval(),
		mode:     "chat",
	}

	updated, _ := m.Update(tea.WindowSizeMsg{Width: 72, Height: 22})
	m = updated.(*Model)
	updated, _ = m.Update(runtimeProgressMsg{event: agent.EngineEvent{Phase: agent.PhaseAct}})
	m = updated.(*Model)
	if got := m.status.Status(); got != "working" {
		t.Fatalf("progress status = %q, want working", got)
	}

	updated, _ = m.Update(agentResponseMsg{
		content:  "ทดสอบสำเร็จ",
		provider: "fixture",
		model:    "model",
		tokens:   3,
		taskID:   "task-test",
		manifest: "agent-folder/output/task-test/manifest.json",
	})
	m = updated.(*Model)

	if got := m.status.Status(); got != "idle" {
		t.Fatalf("result status = %q, want idle", got)
	}
	if len(m.chat.messages) != 2 {
		t.Fatalf("messages = %d, want agent result and manifest", len(m.chat.messages))
	}
	if m.chat.messages[0].Content != "ทดสอบสำเร็จ" {
		t.Fatalf("agent content = %q", m.chat.messages[0].Content)
	}
	if !strings.Contains(m.chat.messages[1].Content, "agent-folder/output/task-test/manifest.json") {
		t.Fatalf("manifest message = %q", m.chat.messages[1].Content)
	}
	if view := m.View(); strings.TrimSpace(view) == "" {
		t.Fatal("headless TUI view is empty")
	}
}
