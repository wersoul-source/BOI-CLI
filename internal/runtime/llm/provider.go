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
	// OnText, when set, asks the Provider to stream and receives the assistant
	// text produced so far, not the latest piece. Cumulative text is safe to
	// show as is after a Router retry restarts a reply, which a delta would
	// duplicate. The whole text is still returned in the response, so a caller
	// that leaves this nil is unaffected, and a Provider that cannot stream
	// simply never calls it.
	OnText func(text string)
}

type Message struct {
	Role    string
	Content string
	// ToolCalls are the native calls an assistant message made.
	ToolCalls []ToolCallRequest
	// ToolCallID links a "tool" role message to the call it answers.
	ToolCallID string
	// ReasoningContent is the reasoning an assistant message came with. It is
	// replayed to the endpoint that produced it, because some endpoints
	// (DeepSeek thinking mode) answer HTTP 400 when it is missing from a later
	// request. It is never set on a message the endpoint did not produce
	// reasoning for, and it is never shown as an answer.
	ReasoningContent string
}

// Fallback records that the Connector or the Runtime took a second route
// because the first did not work. A fallback is never silent: it is carried to
// the result so the user and the trace can see which route was used and why
// (Architecture IV, P1.5).
type Fallback struct {
	// Kind names the route change, such as "native→envelope" or
	// "stream→whole-response".
	Kind string
	// Reason is the evidence: what the endpoint did or said.
	Reason string
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
	// ToolCallDialect is set when the adapter recovered ToolCalls from markup
	// the model wrote into Content ("dsml", "hermes", "qwen-xml") instead of
	// the protocol's tool-call field. Such calls are a compensation: the
	// capability they show is ASSISTED, never NATIVE (ADR-006).
	ToolCallDialect string
	// ReasoningContent is the reasoning the endpoint returned, if any.
	ReasoningContent string
	// Fallbacks are the route changes the adapter made to produce this reply.
	Fallbacks []Fallback
	// Rate-limit info from API headers (0 if unknown)
	RateLimitRemaining int
	RateLimitTotal     int
}

type Provider interface {
	Name() string
	Complete(ctx context.Context, req CompletionRequest) (*CompletionResponse, error)
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
