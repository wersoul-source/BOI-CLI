package llm

import "context"

type CompletionRequest struct {
	Messages    []Message
	MaxTokens   int
	Temperature float64
	// Tools are native tool definitions. Providers that do not implement
	// ToolCaller ignore them, so callers send Tools only when every candidate
	// Provider supports them.
	Tools []ToolDefinition
}

type Message struct {
	Role    string
	Content string
	// ToolCalls are the native calls an assistant message made.
	ToolCalls []ToolCallRequest
	// ToolCallID links a "tool" role message to the call it answers.
	ToolCallID string
}

// ToolDefinition describes one callable tool in a Provider-neutral form.
type ToolDefinition struct {
	Name        string
	Description string
	Parameters  map[string]any
}

// ToolCallRequest is one native tool call proposed by a model. Arguments is
// the raw JSON object text the model produced.
type ToolCallRequest struct {
	ID        string
	Name      string
	Arguments string
}

type CompletionResponse struct {
	Content      string
	ToolCalls    []ToolCallRequest
	InputTokens  int
	OutputTokens int
	Model        string
	Provider     string
	// FinishReason is why the model stopped, in the Provider's own words
	// ("stop", "length", "tool_calls", …). It is empty when the adapter or the
	// endpoint does not report one. "length" tells a truncated reply apart
	// from a wrong one.
	FinishReason string
	// Rate-limit info from API headers (0 if unknown)
	RateLimitRemaining int
	RateLimitTotal     int
}

type Token struct {
	Text  string
	Error error
	Done  bool
}

type Provider interface {
	Name() string
	Complete(ctx context.Context, req CompletionRequest) (*CompletionResponse, error)
	Stream(ctx context.Context, req CompletionRequest) (<-chan Token, error)
}

// ToolCaller is implemented by a Provider whose adapter sends
// CompletionRequest.Tools and returns CompletionResponse.ToolCalls. It states
// protocol support only; whether the model actually uses tools is measured by
// qualification.
type ToolCaller interface {
	SupportsNativeTools() bool
}

// SupportsNativeTools reports whether p implements ToolCaller and enables it.
func SupportsNativeTools(p Provider) bool {
	caller, ok := p.(ToolCaller)
	return ok && caller.SupportsNativeTools()
}
