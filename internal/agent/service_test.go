package agent

import (
	"context"
	"os"
	"strconv"
	"strings"
	"testing"

	"github.com/boi-family/boi-cli/internal/persona"
	llm "github.com/boi-family/boi-cli/internal/provider"
	"github.com/boi-family/boi-cli/internal/workspace"
)

type captureProvider struct {
	request llm.CompletionRequest
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
