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
		client:  &http.Client{},
	}
}

func (p *OpenAIProvider) Name() string {
	return p.name
}

type openAIMessage struct {
	Role    string `json:"role"`
	Content string `json:"content"`
}

type openAIRequest struct {
	Model       string          `json:"model"`
	Messages    []openAIMessage `json:"messages"`
	MaxTokens   int             `json:"max_tokens,omitempty"`
	Temperature float64         `json:"temperature,omitempty"`
	Stream      bool            `json:"stream"`
}

type openAIResponse struct {
	Choices []struct {
		Message struct {
			Content string `json:"content"`
		} `json:"message"`
	} `json:"choices"`
	Usage struct {
		PromptTokens     int `json:"prompt_tokens"`
		CompletionTokens int `json:"completion_tokens"`
	} `json:"usage"`
	Model string `json:"model"`
}

func (p *OpenAIProvider) Complete(ctx context.Context, req llm.CompletionRequest) (*llm.CompletionResponse, error) {
	messages := make([]openAIMessage, len(req.Messages))
	for i, m := range req.Messages {
		messages[i] = openAIMessage{Role: m.Role, Content: m.Content}
	}

	body := openAIRequest{
		Model:       p.model,
		Messages:    messages,
		MaxTokens:   req.MaxTokens,
		Temperature: req.Temperature,
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
	httpReq.Header.Set("Authorization", "Bearer "+p.apiKey)

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

	var result openAIResponse
	if err := json.Unmarshal(respBody, &result); err != nil {
		return nil, fmt.Errorf("provider %s: %w", p.name, err)
	}

	if len(result.Choices) == 0 {
		return nil, fmt.Errorf("provider %s: no choices in response", p.name)
	}

	rateLimitRemaining, rateLimitTotal := parseRateLimitHeaders(resp.Header)

	return &llm.CompletionResponse{
		Content:            result.Choices[0].Message.Content,
		InputTokens:        result.Usage.PromptTokens,
		OutputTokens:       result.Usage.CompletionTokens,
		Model:              result.Model,
		Provider:           p.name,
		RateLimitRemaining: rateLimitRemaining,
		RateLimitTotal:     rateLimitTotal,
	}, nil
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
