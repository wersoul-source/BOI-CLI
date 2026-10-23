package tui

import (
	"strings"
	"testing"

	"github.com/boi-family/boi-cli/internal/app"
	llmfactory "github.com/boi-family/boi-cli/internal/service/provider/factory"
)

func TestEveryListedCommandIsRecognised(t *testing.T) {
	t.Parallel()

	m := &Model{status: NewStatus("แก้ว", "none", nil), root: "/work"}
	for _, cmd := range availableCommands {
		if isWorkspaceCommand(cmd) {
			continue
		}
		got, _ := m.handleSlashCommand(cmd)
		if strings.Contains(got, "not recognized") {
			t.Errorf("%s is offered in the palette but %q", cmd, got)
		}
	}
}

func TestModelAndConfigWithoutProviderGuide(t *testing.T) {
	t.Parallel()

	m := &Model{status: NewStatus("แก้ว", "none", nil), root: "/work"}
	for _, cmd := range []string{"/model", "/model foo", "/config", "/provider"} {
		got, _ := m.handleSlashCommand(cmd)
		if !strings.Contains(got, "quickstart") || !strings.Contains(got, "/work") && cmd != "/provider" && cmd != "/model" && cmd != "/model foo" {
			t.Errorf("%s = %q, want a quickstart hint", cmd, got)
		}
	}
}

func TestConfiguredButUnqualifiedIsNotCalledMissing(t *testing.T) {
	t.Parallel()

	m := &Model{
		status:     NewStatus("แก้ว", "unqualified", nil),
		configured: []llmfactory.ConfiguredProvider{{Name: "openai", Model: "m1"}},
	}
	got := m.noProviderMsg()
	if !strings.Contains(got, "openai/m1") || strings.Contains(got, "No model is configured") {
		t.Fatalf("message = %q", got)
	}
}

func TestModelShowsQualifiedModel(t *testing.T) {
	t.Parallel()

	m := &Model{
		status: NewStatus("แก้ว", "x", nil),
		root:   "/work",
		agent: &app.Agent{Qualified: []llmfactory.ConfiguredProvider{
			{Name: "openai", Model: "qwen-coder", BaseURL: "https://example.test/v1"},
		}},
	}
	got, _ := m.handleSlashCommand("/model")
	for _, want := range []string{"qwen-coder", "openai", "https://example.test/v1"} {
		if !strings.Contains(got, want) {
			t.Errorf("/model = %q, missing %q", got, want)
		}
	}
	if strings.Contains(got, "Model set to") {
		t.Fatalf("/model claims to switch: %q", got)
	}
}
