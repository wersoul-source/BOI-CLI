package adapters

import (
	"bytes"
	"context"
	"encoding/json"
	"fmt"
	"net/http"
	"strings"

	llm "github.com/boi-family/boi-cli/internal/runtime/llm"
)

type OpenAIProvider struct {
	name    string
	apiKey  string
	baseURL string
	model   string
	client  *http.Client
}

func NewOpenAIProvider(name, apiKey, baseURL, model string) llm.Provider {
	if baseURL == "" {
		baseURL = "https://api.openai.com/v1"
	}
	baseURL = strings.TrimRight(baseURL, "/")
	return &OpenAIProvider{
		name:    name,
		apiKey:  apiKey,
		baseURL: baseURL,
		model:   model,
		client:  newHTTPClient(),
	}
}

func (p *OpenAIProvider) Name() string {
	return p.name
}

// SupportsNativeTools implements llm.ToolCaller: the Chat Completions
// protocol carries "tools" and "tool_calls".
func (p *OpenAIProvider) SupportsNativeTools() bool { return true }

type openAIMessage struct {
	Role string `json:"role"`
	// Content is null on an assistant message that only calls tools.
	Content    *string          `json:"content"`
	ToolCalls  []openAIToolCall `json:"tool_calls,omitempty"`
	ToolCallID string           `json:"tool_call_id,omitempty"`
}

type openAIToolCall struct {
	ID       string `json:"id"`
	Type     string `json:"type"`
	Function struct {
		Name      string `json:"name"`
		Arguments string `json:"arguments"`
	} `json:"function"`
}

type openAITool struct {
	Type     string `json:"type"`
	Function struct {
		Name        string         `json:"name"`
		Description string         `json:"description,omitempty"`
		Parameters  map[string]any `json:"parameters"`
	} `json:"function"`
}

type openAIRequest struct {
	Model     string          `json:"model"`
	Messages  []openAIMessage `json:"messages"`
	Tools     []openAITool    `json:"tools,omitempty"`
	MaxTokens int             `json:"max_tokens,omitempty"`
	// Temperature is a pointer so 0 (deterministic probes) is sent, not
	// dropped in favor of the provider's default.
	Temperature *float64 `json:"temperature,omitempty"`
	Stream      bool     `json:"stream"`
}

type openAIResponse struct {
	Choices []struct {
		Message struct {
			Content   string           `json:"content"`
			ToolCalls []openAIToolCall `json:"tool_calls"`
		} `json:"message"`
	} `json:"choices"`
	Usage struct {
		PromptTokens     int `json:"prompt_tokens"`
		CompletionTokens int `json:"completion_tokens"`
	} `json:"usage"`
	Model string `json:"model"`
}

func (p *OpenAIProvider) Complete(ctx context.Context, req llm.CompletionRequest) (*llm.CompletionResponse, error) {
	body := openAIRequest{
		Model:       p.model,
		Messages:    toOpenAIMessages(req.Messages),
		Tools:       toOpenAITools(req.Tools),
		MaxTokens:   req.MaxTokens,
		Temperature: &req.Temperature,
		Stream:      false,
	}

	jsonBody, err := json.Marshal(body)
	if err != nil {
		return nil, fmt.Errorf("provider %s: %w", p.name, err)
	}

	httpReq, err := http.NewRequestWithContext(ctx, "POST", p.baseURL+"/chat/completions", bytes.NewReader(jsonBody))
	if err != nil {
		return nil, fmt.Errorf("provider %s: %w", p.name, err)
	}
	httpReq.Header.Set("Content-Type", "application/json")
	if p.apiKey != "" {
		httpReq.Header.Set("Authorization", "Bearer "+p.apiKey)
	}

	resp, err := p.client.Do(httpReq)
	if err != nil {
		return nil, fmt.Errorf("provider %s: %w", p.name, err)
	}
	defer resp.Body.Close()

	respBody, err := readBody(resp.Body)
	if err != nil {
		return nil, fmt.Errorf("provider %s: %w", p.name, err)
	}

	if resp.StatusCode != http.StatusOK {
		return nil, llm.NewHTTPError(p.name, resp.StatusCode, string(respBody), resp.Header)
	}

	var result openAIResponse
	if err := json.Unmarshal(respBody, &result); err != nil {
		return nil, fmt.Errorf("provider %s: %w", p.name, err)
	}

	if len(result.Choices) == 0 {
		return nil, fmt.Errorf("provider %s: no choices in response", p.name)
	}

	rateLimitRemaining, rateLimitTotal := parseRateLimitHeaders(resp.Header)

	var calls []llm.ToolCallRequest
	for _, call := range result.Choices[0].Message.ToolCalls {
		calls = append(calls, llm.ToolCallRequest{ID: call.ID, Name: call.Function.Name, Arguments: call.Function.Arguments})
	}

	return &llm.CompletionResponse{
		Content:            result.Choices[0].Message.Content,
		ToolCalls:          calls,
		InputTokens:        result.Usage.PromptTokens,
		OutputTokens:       result.Usage.CompletionTokens,
		Model:              result.Model,
		Provider:           p.name,
		RateLimitRemaining: rateLimitRemaining,
		RateLimitTotal:     rateLimitTotal,
	}, nil
}

func toOpenAIMessages(messages []llm.Message) []openAIMessage {
	out := make([]openAIMessage, len(messages))
	for i, m := range messages {
		content := m.Content
		out[i] = openAIMessage{Role: m.Role, Content: &content, ToolCallID: m.ToolCallID}
		if len(m.ToolCalls) > 0 && content == "" {
			out[i].Content = nil
		}
		for _, call := range m.ToolCalls {
			var item openAIToolCall
			item.ID, item.Type = call.ID, "function"
			item.Function.Name, item.Function.Arguments = call.Name, call.Arguments
			out[i].ToolCalls = append(out[i].ToolCalls, item)
		}
	}
	return out
}

func toOpenAITools(tools []llm.ToolDefinition) []openAITool {
	out := make([]openAITool, 0, len(tools))
	for _, tool := range tools {
		var item openAITool
		item.Type = "function"
		item.Function.Name, item.Function.Description, item.Function.Parameters = tool.Name, tool.Description, tool.Parameters
		out = append(out, item)
	}
	return out
}

func parseRateLimitHeaders(h http.Header) (remaining, total int) {
	if v := h.Get("x-ratelimit-remaining-tokens"); v != "" {
		fmt.Sscanf(v, "%d", &remaining)
	}
	if v := h.Get("x-ratelimit-limit-tokens"); v != "" {
		fmt.Sscanf(v, "%d", &total)
	}
	// Try requests-based headers as fallback
	if remaining == 0 && total == 0 {
		if v := h.Get("x-ratelimit-remaining-requests"); v != "" {
			fmt.Sscanf(v, "%d", &remaining)
		}
		if v := h.Get("x-ratelimit-limit-requests"); v != "" {
			fmt.Sscanf(v, "%d", &total)
		}
	}
	return
}

func (p *OpenAIProvider) Stream(ctx context.Context, req llm.CompletionRequest) (<-chan llm.Token, error) {
	resp, err := p.Complete(ctx, req)
	if err != nil {
		return nil, err
	}

	ch := make(chan llm.Token, 1)
	go func() {
		defer close(ch)
		ch <- llm.Token{Text: resp.Content}
		ch <- llm.Token{Done: true}
	}()
	return ch, nil
}
