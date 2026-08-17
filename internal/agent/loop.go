package agent

import (
	"context"
	"fmt"
	"log/slog"
	"time"

	"github.com/boi-family/boi-cli/internal/memory"
	"github.com/boi-family/boi-cli/internal/persona"
	llm "github.com/boi-family/boi-cli/internal/provider"
)

type Loop struct {
	state     *AgentState
	persona   *persona.Persona
	providers []llm.Provider
	hook      *memory.MemoryHook
	MaxSteps  int
	logger    *slog.Logger
}

func NewLoop(persona *persona.Persona, providers []llm.Provider, hook *memory.MemoryHook) *Loop {
	return &Loop{
		persona:   persona,
		providers: providers,
		hook:      hook,
		MaxSteps:  15,
		logger:    slog.Default(),
	}
}

func (l *Loop) Run(ctx context.Context, query string) (*AgentResult, error) {
	l.state = &AgentState{
		ID:          fmt.Sprintf("agent_%d", time.Now().UnixNano()),
		PersonaName: l.persona.Name,
		Status:      "thinking",
		Task:        query,
		StartedAt:   time.Now(),
	}

	if l.hook != nil {
		contextBlock := l.hook.BeforeTurn(query)
		if contextBlock != "" {
			l.state.MemoryUsed = 1
		}
	}

	systemPrompt := l.buildSystemPrompt()

	messages := []llm.Message{
		{Role: "system", Content: systemPrompt},
		{Role: "user", Content: query},
	}

	for i := 0; i < l.MaxSteps; i++ {
		select {
		case <-ctx.Done():
			return nil, ctx.Err()
		default:
		}

		resp, err := l.callLLM(ctx, messages)
		if err != nil {
			l.state.Status = "error"
			return nil, fmt.Errorf("LLM failed at step %d: %w", i, err)
		}

		if resp.Content != "" && !l.needsTools(resp.Content) {
			if l.hook != nil {
				l.hook.AfterTurn(query, resp.Content)
			}

			l.state.Status = "done"
			step := AgentStep{
				Number:   i + 1,
				Thought:  "Task complete",
				Action:   "respond",
				Result:   resp.Content,
				Success:  true,
				Duration: time.Since(l.state.StartedAt),
			}
			l.state.Steps = append(l.state.Steps, step)

			return &AgentResult{
				Response: resp.Content,
				Steps:    i + 1,
				Tokens:   resp.InputTokens + resp.OutputTokens,
				Duration: time.Since(l.state.StartedAt),
				Provider: resp.Provider,
				Model:    resp.Model,
			}, nil
		}

		step := AgentStep{
			Number:  i + 1,
			Thought: "Processing...",
			Action:  "llm-call",
			Result:  resp.Content,
			Success: true,
		}
		l.state.Steps = append(l.state.Steps, step)

		messages = append(messages, llm.Message{Role: "assistant", Content: resp.Content})
	}

	return nil, fmt.Errorf("exceeded max steps (%d)", l.MaxSteps)
}

func (l *Loop) buildSystemPrompt() string {
	prompt := l.persona.SystemPrompt
	if prompt == "" {
		prompt = "You are a helpful AI assistant."
	}
	return prompt
}

func (l *Loop) callLLM(ctx context.Context, messages []llm.Message) (*llm.CompletionResponse, error) {
	if len(l.providers) == 0 {
		return l.simulatedResponse(messages)
	}
	router := llm.NewRouter(l.providers)
	return router.Complete(ctx, llm.CompletionRequest{
		Messages:    messages,
		MaxTokens:   4096,
		Temperature: l.persona.Temperature,
	})
}

func (l *Loop) simulatedResponse(messages []llm.Message) (*llm.CompletionResponse, error) {
	var userQuery string
	for _, m := range messages {
		if m.Role == "user" {
			userQuery = m.Content
		}
	}
	return &llm.CompletionResponse{
		Content:      fmt.Sprintf("[Simulated %s] I received your query: %s\n\nNo LLM providers configured. Set PSC_1_NAME, PSC_1_API_KEY, PSC_1_MODEL in .env to enable AI responses.", l.persona.Name, userQuery),
		InputTokens:  0,
		OutputTokens: 0,
		Model:        "simulated",
		Provider:     "none",
	}, nil
}

func (l *Loop) needsTools(content string) bool {
	return len(content) < 10
}
