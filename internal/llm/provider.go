package llm

import "context"

type CompletionRequest struct {
	Messages    []Message
	MaxTokens   int
	Temperature float64
}

type Message struct {
	Role    string
	Content string
}

type CompletionResponse struct {
	Content      string
	InputTokens  int
	OutputTokens int
	Model        string
	Provider     string
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
