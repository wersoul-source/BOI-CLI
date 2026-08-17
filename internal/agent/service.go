package agent

import (
	"context"
	"errors"
	"fmt"
	"strings"
	"sync"
	"time"

	"github.com/boi-family/boi-cli/internal/memory"
	"github.com/boi-family/boi-cli/internal/persona"
	llm "github.com/boi-family/boi-cli/internal/provider"
	"github.com/boi-family/boi-cli/internal/workspace"
)

var ErrNoProvider = errors.New("no AI providers configured")

// Service is the shared single-agent entry point used by interactive
// transports. Tool execution is intentionally not enabled until approval state
// is part of the TUI lifecycle.
type Service struct {
	mu      sync.RWMutex
	persona *persona.Persona
	router  *llm.Router
	memory  *memory.MemoryHook
	sandbox *workspace.Sandbox
}

func NewService(
	activePersona *persona.Persona,
	router *llm.Router,
	memoryHook *memory.MemoryHook,
	sandbox *workspace.Sandbox,
) *Service {
	if activePersona == nil {
		activePersona = persona.DefaultPersona()
	}
	return &Service{
		persona: activePersona,
		router:  router,
		memory:  memoryHook,
		sandbox: sandbox,
	}
}

func (s *Service) SetPersona(activePersona *persona.Persona) {
	if activePersona == nil {
		return
	}
	s.mu.Lock()
	s.persona = activePersona
	s.mu.Unlock()
}

func (s *Service) Run(ctx context.Context, query string) (*AgentResult, error) {
	query = strings.TrimSpace(query)
	if query == "" {
		return nil, fmt.Errorf("agent query is empty")
	}
	if s.router == nil || s.router.ProviderCount() == 0 {
		return nil, ErrNoProvider
	}

	s.mu.RLock()
	activePersona := *s.persona
	s.mu.RUnlock()

	startedAt := time.Now()
	systemPrompt := buildServicePrompt(&activePersona, s.sandbox)
	if s.memory != nil {
		if recalled := s.memory.BeforeTurn(query); recalled != "" {
			systemPrompt += "\n\n" + recalled
		}
	}

	maxTokens := activePersona.MaxTokens
	if maxTokens <= 0 {
		maxTokens = 4096
	}
	response, err := s.router.Complete(ctx, llm.CompletionRequest{
		Messages: []llm.Message{
			{Role: "system", Content: systemPrompt},
			{Role: "user", Content: query},
		},
		MaxTokens:   maxTokens,
		Temperature: activePersona.Temperature,
	})
	if err != nil {
		return nil, fmt.Errorf("agent provider call: %w", err)
	}

	if s.memory != nil {
		s.memory.AfterTurn(query, response.Content)
	}

	return &AgentResult{
		Response: response.Content,
		Steps:    1,
		Tokens:   response.InputTokens + response.OutputTokens,
		Duration: time.Since(startedAt),
		Provider: response.Provider,
		Model:    response.Model,
	}, nil
}

func buildServicePrompt(activePersona *persona.Persona, sandbox *workspace.Sandbox) string {
	prompt := activePersona.SystemPrompt
	if strings.TrimSpace(prompt) == "" {
		prompt = "You are a helpful AI assistant."
	}
	if sandbox == nil {
		return prompt
	}
	return prompt + fmt.Sprintf(`

## Workspace boundary
The host workspace root is %q.
Do not claim that you read, changed, or executed anything unless a host tool result proves it.
Filesystem access outside this workspace is forbidden.`, sandbox.Root())
}
