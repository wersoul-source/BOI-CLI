package agent

import (
	"context"
	"fmt"
	"strings"

	"github.com/boi-family/boi-cli/internal/memory"
	"github.com/boi-family/boi-cli/internal/persona"
	llm "github.com/boi-family/boi-cli/internal/provider"
)

// Loop preserves the original CLI-facing API while delegating lifecycle and
// budgets to Engine. New transports should use Service so capability and
// approval policy remains centralized.
type Loop struct {
	persona   *persona.Persona
	providers []llm.Provider
	hook      *memory.MemoryHook
	MaxSteps  int
}

func NewLoop(activePersona *persona.Persona, providers []llm.Provider, hook *memory.MemoryHook) *Loop {
	if activePersona == nil {
		activePersona = persona.DefaultPersona()
	}
	return &Loop{
		persona:   activePersona,
		providers: providers,
		hook:      hook,
		MaxSteps:  15,
	}
}

func (l *Loop) Run(ctx context.Context, query string) (*AgentResult, error) {
	query = strings.TrimSpace(query)
	if query == "" {
		return nil, fmt.Errorf("agent query is empty")
	}
	systemPrompt := l.persona.SystemPrompt
	if strings.TrimSpace(systemPrompt) == "" {
		systemPrompt = "You are a helpful AI assistant."
	}
	if l.hook != nil {
		if recalled := l.hook.BeforeTurn(query); recalled != "" {
			systemPrompt += "\n\n" + recalled
		}
	}

	router := llm.NewRouter(l.providers)
	decider := decisionFunc(func(decideCtx context.Context, _ DecisionInput) (Decision, error) {
		var response *llm.CompletionResponse
		var err error
		if len(l.providers) == 0 {
			response = l.simulatedResponse(query)
		} else {
			response, err = router.Complete(decideCtx, llm.CompletionRequest{
				Messages: []llm.Message{
					{Role: "system", Content: systemPrompt},
					{Role: "user", Content: query},
				},
				MaxTokens:   4096,
				Temperature: l.persona.Temperature,
			})
		}
		if err != nil {
			return Decision{}, err
		}
		return Decision{
			Kind:     DecisionRespond,
			Response: response.Content,
			Provider: response.Provider,
			Model:    response.Model,
			Usage: Usage{
				InputTokens:   response.InputTokens,
				OutputTokens:  response.OutputTokens,
				ProviderCalls: 1,
			},
		}, nil
	})
	limits := DefaultEngineLimits()
	limits.MaxSteps = l.MaxSteps
	engine := &Engine{Decider: decider, Limits: limits}
	result, err := engine.Run(ctx, query)
	if err != nil {
		return nil, err
	}
	if result.StopReason != StopCompleted {
		if ctx.Err() != nil {
			return nil, ctx.Err()
		}
		return nil, fmt.Errorf("agent stopped (%s): %s", result.StopReason, result.Error)
	}
	if l.hook != nil {
		l.hook.AfterTurn(query, result.Response)
	}
	return result, nil
}

func (l *Loop) simulatedResponse(query string) *llm.CompletionResponse {
	return &llm.CompletionResponse{
		Content: fmt.Sprintf(
			"[Simulated %s] I received your query: %s\n\nNo LLM providers configured. Set PSC_1_NAME, PSC_1_API_KEY, PSC_1_MODEL in .env to enable AI responses.",
			l.persona.Name,
			query,
		),
		Model:    "simulated",
		Provider: "none",
	}
}
