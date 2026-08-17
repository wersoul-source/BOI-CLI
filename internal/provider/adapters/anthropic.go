package adapters

import (
	"bytes"
	"context"
	"encoding/json"
	"fmt"
	"io"
	"net/http"
	"strings"

	llm "github.com/boi-family/boi-cli/internal/provider"
)

type AnthropicProvider struct {
	name    string
	apiKey  string
	baseURL string
	model   string
	client  *http.Client
}

func NewAnthropicProvider(name, apiKey, baseURL, model string) llm.Provider {
	if baseURL == "" {
		baseURL = "https://api.anthropic.com/v1"
	}
	baseURL = strings.TrimRight(baseURL, "/")
	return &AnthropicProvider{
		name:    name,
		apiKey:  apiKey,
		baseURL: baseURL,
		model:   model,
		client:  &http.Client{},
	}
}

func (p *AnthropicProvider) Name() string {
	return p.name
}

type anthropicContent struct {
	Type string `json:"type"`
	Text string `json:"text,omitempty"`
}

type anthropicMessage struct {
	Role    string             `json:"role"`
	Content []anthropicContent `json:"content"`
}

type anthropicRequest struct {
	Model       string             `json:"model"`
	MaxTokens   int                `json:"max_tokens"`
	Temperature float64            `json:"temperature,omitempty"`
	System      string             `json:"system,omitempty"`
	Messages    []anthropicMessage `json:"messages"`
}

type anthropicResponse struct {
	Content []anthropicContent `json:"content"`
	Usage   struct {
		InputTokens  int `json:"input_tokens"`
		OutputTokens int `json:"output_tokens"`
	} `json:"usage"`
	Model string `json:"model"`
}

func (p *AnthropicProvider) Complete(ctx context.Context, req llm.CompletionRequest) (*llm.CompletionResponse, error) {
	var system string
	var messages []anthropicMessage

	for _, m := range req.Messages {
		if strings.EqualFold(m.Role, "system") {
			system = m.Content
			continue
		}
		role := m.Role
		if role == "user" {
			role = "user"
		} else if role == "assistant" {
			role = "assistant"
		}
		messages = append(messages, anthropicMessage{
			Role: role,
			Content: []anthropicContent{
				{Type: "text", Text: m.Content},
			},
		})
	}

	maxTokens := req.MaxTokens
	if maxTokens == 0 {
		maxTokens = 1024
	}

	body := anthropicRequest{
		Model:       p.model,
		MaxTokens:   maxTokens,
		Temperature: req.Temperature,
		System:      system,
		Messages:    messages,
	}

	jsonBody, err := json.Marshal(body)
	if err != nil {
		return nil, fmt.Errorf("provider %s: %w", p.name, err)
	}

	httpReq, err := http.NewRequestWithContext(ctx, "POST", p.baseURL+"/messages", bytes.NewReader(jsonBody))
	if err != nil {
		return nil, fmt.Errorf("provider %s: %w", p.name, err)
	}
	httpReq.Header.Set("Content-Type", "application/json")
	httpReq.Header.Set("x-api-key", p.apiKey)
	httpReq.Header.Set("anthropic-version", "2023-06-01")

	resp, err := p.client.Do(httpReq)
	if err != nil {
		return nil, fmt.Errorf("provider %s: %w", p.name, err)
	}
	defer resp.Body.Close()

	respBody, err := io.ReadAll(resp.Body)
	if err != nil {
		return nil, fmt.Errorf("provider %s: %w", p.name, err)
	}

	if resp.StatusCode != http.StatusOK {
		return nil, fmt.Errorf("provider %s: status %d: %s", p.name, resp.StatusCode, string(respBody))
	}

	var result anthropicResponse
	if err := json.Unmarshal(respBody, &result); err != nil {
		return nil, fmt.Errorf("provider %s: %w", p.name, err)
	}

	if len(result.Content) == 0 {
		return nil, fmt.Errorf("provider %s: no content in response", p.name)
	}

	return &llm.CompletionResponse{
		Content:      result.Content[0].Text,
		InputTokens:  result.Usage.InputTokens,
		OutputTokens: result.Usage.OutputTokens,
		Model:        result.Model,
		Provider:     p.name,
	}, nil
}

func (p *AnthropicProvider) Stream(ctx context.Context, req llm.CompletionRequest) (<-chan llm.Token, error) {
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
