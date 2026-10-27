package adapters

import (
	"bufio"
	"bytes"
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"io"
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
	Temperature   *float64             `json:"temperature,omitempty"`
	Stream        bool                 `json:"stream"`
	StreamOptions *openAIStreamOptions `json:"stream_options,omitempty"`
}

type openAIStreamOptions struct {
	IncludeUsage bool `json:"include_usage"`
}

// openAIStreamChunk is one server-sent event of a streamed completion. Text
// arrives in delta.content; a tool call arrives in pieces that share an index,
// with its arguments split across any number of chunks.
type openAIStreamChunk struct {
	Choices []struct {
		Delta struct {
			Content   string `json:"content"`
			ToolCalls []struct {
				Index    int    `json:"index"`
				ID       string `json:"id"`
				Function struct {
					Name      string `json:"name"`
					Arguments string `json:"arguments"`
				} `json:"function"`
			} `json:"tool_calls"`
		} `json:"delta"`
		FinishReason string `json:"finish_reason"`
	} `json:"choices"`
	Usage *struct {
		PromptTokens     int `json:"prompt_tokens"`
		CompletionTokens int `json:"completion_tokens"`
	} `json:"usage"`
	Model string `json:"model"`
}

type openAIResponse struct {
	Choices []struct {
		Message struct {
			Content   string           `json:"content"`
			ToolCalls []openAIToolCall `json:"tool_calls"`
		} `json:"message"`
		FinishReason string `json:"finish_reason"`
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
	if req.OnText != nil {
		return p.stream(ctx, req, body)
	}

	resp, err := p.send(ctx, body)
	if err != nil {
		return nil, err
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
		FinishReason:       result.Choices[0].FinishReason,
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

// send posts one request and returns the open response. The caller closes the
// body.
func (p *OpenAIProvider) send(ctx context.Context, body openAIRequest) (*http.Response, error) {
	jsonBody, err := json.Marshal(body)
	if err != nil {
		return nil, fmt.Errorf("provider %s: %w", p.name, err)
	}
	httpReq, err := http.NewRequestWithContext(ctx, "POST", p.baseURL+"/chat/completions", bytes.NewReader(jsonBody))
	if err != nil {
		return nil, fmt.Errorf("provider %s: %w", p.name, err)
	}
	httpReq.Header.Set("Content-Type", "application/json")
	if body.Stream {
		httpReq.Header.Set("Accept", "text/event-stream")
	}
	if p.apiKey != "" {
		httpReq.Header.Set("Authorization", "Bearer "+p.apiKey)
	}
	resp, err := p.client.Do(httpReq)
	if err != nil {
		return nil, fmt.Errorf("provider %s: %w", p.name, err)
	}
	return resp, nil
}

// stream reads a server-sent event stream and reports the text as it arrives.
// It returns the same response a non-streamed call would, so nothing above it
// has to know which path ran.
func (p *OpenAIProvider) stream(ctx context.Context, req llm.CompletionRequest, body openAIRequest) (*llm.CompletionResponse, error) {
	body.Stream = true
	body.StreamOptions = &openAIStreamOptions{IncludeUsage: true}
	resp, err := p.send(ctx, body)
	if err != nil {
		return nil, err
	}
	defer resp.Body.Close()
	if resp.StatusCode != http.StatusOK {
		respBody, _ := readBody(resp.Body)
		return nil, llm.NewHTTPError(p.name, resp.StatusCode, string(respBody), resp.Header)
	}

	result := &llm.CompletionResponse{Provider: p.name, Model: p.model}
	var content strings.Builder
	calls := newToolCallAccumulator()
	reader := bufio.NewReaderSize(resp.Body, 64<<10)
	read := 0
	for {
		line, readErr := reader.ReadString('\n')
		if read += len(line); read > maxResponseBytes {
			return nil, fmt.Errorf("provider %s: stream exceeds %d bytes", p.name, maxResponseBytes)
		}
		payload, isData := strings.CutPrefix(strings.TrimRight(line, "\r\n"), "data:")
		if isData {
			payload = strings.TrimSpace(payload)
			if payload == "[DONE]" {
				break
			}
			var chunk openAIStreamChunk
			// A comment, a keep-alive or an event this adapter does not model
			// is not an error: the stream carries on.
			if json.Unmarshal([]byte(payload), &chunk) == nil {
				if chunk.Model != "" {
					result.Model = chunk.Model
				}
				if chunk.Usage != nil {
					result.InputTokens, result.OutputTokens = chunk.Usage.PromptTokens, chunk.Usage.CompletionTokens
				}
				for _, choice := range chunk.Choices {
					if choice.FinishReason != "" {
						result.FinishReason = choice.FinishReason
					}
					for _, call := range choice.Delta.ToolCalls {
						calls.add(call.Index, call.ID, call.Function.Name, call.Function.Arguments)
					}
					if choice.Delta.Content != "" {
						content.WriteString(choice.Delta.Content)
						req.OnText(content.String())
					}
				}
			}
		}
		if readErr != nil {
			if errors.Is(readErr, io.EOF) {
				break
			}
			if ctxErr := ctx.Err(); ctxErr != nil {
				return nil, ctxErr
			}
			return nil, fmt.Errorf("provider %s: read stream: %w", p.name, readErr)
		}
	}

	result.Content = content.String()
	result.ToolCalls = calls.requests()
	result.RateLimitRemaining, result.RateLimitTotal = parseRateLimitHeaders(resp.Header)
	return result, nil
}

// toolCallAccumulator rebuilds streamed tool calls. Each call keeps the index
// it was announced with, and its arguments are appended in arrival order.
type toolCallAccumulator struct {
	order     []int
	byIndex   map[int]*llm.ToolCallRequest
	arguments map[int]*strings.Builder
}

func newToolCallAccumulator() *toolCallAccumulator {
	return &toolCallAccumulator{byIndex: map[int]*llm.ToolCallRequest{}, arguments: map[int]*strings.Builder{}}
}

func (a *toolCallAccumulator) add(index int, id, name, arguments string) {
	call, seen := a.byIndex[index]
	if !seen {
		call = &llm.ToolCallRequest{}
		a.byIndex[index] = call
		a.arguments[index] = &strings.Builder{}
		a.order = append(a.order, index)
	}
	if id != "" {
		call.ID = id
	}
	if name != "" {
		call.Name = name
	}
	a.arguments[index].WriteString(arguments)
}

func (a *toolCallAccumulator) requests() []llm.ToolCallRequest {
	var out []llm.ToolCallRequest
	for _, index := range a.order {
		call := *a.byIndex[index]
		call.Arguments = a.arguments[index].String()
		out = append(out, call)
	}
	return out
}
