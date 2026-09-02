package tui

import (
	"strings"
	"testing"
)

func TestStatusShowsAgentNameNotSelectablePersona(t *testing.T) {
	t.Parallel()

	status := NewStatus("แก้ว", "test", nil)
	status.SetWidth(120)
	view := status.View()
	if !strings.Contains(view, "Agent: แก้ว") {
		t.Fatalf("status does not show Agent name: %q", view)
	}
	if strings.Contains(view, "Persona:") {
		t.Fatalf("status still presents Persona as selectable identity: %q", view)
	}
}

func TestPersonaCommandReportsCoreInvariant(t *testing.T) {
	t.Parallel()

	m := &Model{status: NewStatus("แก้ว", "test", nil)}
	got, clear := m.handleSlashCommand("/persona")
	if clear {
		t.Fatal("/persona unexpectedly clears chat")
	}
	if got != "Core Persona: boi (fixed)\nAgent: แก้ว" {
		t.Fatalf("/persona response = %q", got)
	}
}

func TestSplashShowsAgentAndCorePersona(t *testing.T) {
	t.Parallel()

	splash := NewSplash("C:/workspace", "แก้ว", "boi", 1, 0, 0, "", "test")
	view := splash.View()
	for _, want := range []string{"Agent: แก้ว", "Core Persona: boi"} {
		if !strings.Contains(view, want) {
			t.Fatalf("splash does not contain %q", want)
		}
	}
}
