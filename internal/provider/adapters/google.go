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

type GoogleProvider struct {
	name    string
	apiKey  string
	baseURL string
	model   string
	client  *http.Client
}

func NewGoogleProvider(name, apiKey, baseURL, model string) llm.Provider {
	if baseURL == "" {
		baseURL = "https://generativelanguage.googleapis.com/v1beta"
	}
	baseURL = strings.TrimRight(baseURL, "/")
	return &GoogleProvider{
		name:    name,
		apiKey:  apiKey,
		baseURL: baseURL,
		model:   model,
		client:  &http.Client{},
	}
}

func (p *GoogleProvider) Name() string {
	return p.name
}

type googlePart struct {
	Text string `json:"text"`
}

type googleContent struct {
	Role  string       `json:"role,omitempty"`
	Parts []googlePart `json:"parts"`
}

type googleSystemInstruction struct {
	Parts []googlePart `json:"parts"`
}

type googleGenerationConfig struct {
	MaxOutputTokens int     `json:"maxOutputTokens,omitempty"`
	Temperature     float64 `json:"temperature,omitempty"`
}

type googleRequest struct {
	SystemInstruction *googleSystemInstruction `json:"system_instruction,omitempty"`
	Contents          []googleContent          `json:"contents"`
	GenerationConfig  *googleGenerationConfig  `json:"generationConfig,omitempty"`
}

type googleResponse struct {
	Candidates []struct {
		Content struct {
			Parts []googlePart `json:"parts"`
		} `json:"content"`
	} `json:"candidates"`
	UsageMetadata struct {
		PromptTokenCount     int `json:"promptTokenCount"`
		CandidatesTokenCount int `json:"candidatesTokenCount"`
	} `json:"usageMetadata"`
}

func (p *GoogleProvider) Complete(ctx context.Context, req llm.CompletionRequest) (*llm.CompletionResponse, error) {
	var systemParts []googlePart
	var contents []googleContent

	for _, m := range req.Messages {
		if m.Role == "system" {
			systemParts = append(systemParts, googlePart{Text: m.Content})
		} else {
			role := m.Role
			if role == "assistant" {
				role = "model"
			}
			contents = append(contents, googleContent{
				Role:  role,
				Parts: []googlePart{{Text: m.Content}},
			})
		}
	}

	body := googleRequest{
		Contents: contents,
		GenerationConfig: &googleGenerationConfig{
			MaxOutputTokens: req.MaxTokens,
			Temperature:     req.Temperature,
		},
	}
	if len(systemParts) > 0 {
		body.SystemInstruction = &googleSystemInstruction{Parts: systemParts}
	}

	jsonBody, err := json.Marshal(body)
	if err != nil {
		return nil, fmt.Errorf("provider %s: %w", p.name, err)
	}

	url := fmt.Sprintf("%s/models/%s:generateContent?key=%s", p.baseURL, p.model, p.apiKey)
	httpReq, err := http.NewRequestWithContext(ctx, "POST", url, bytes.NewReader(jsonBody))
	if err != nil {
		return nil, fmt.Errorf("provider %s: %w", p.name, err)
	}
	httpReq.Header.Set("Content-Type", "application/json")

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
		return nil, fmt.Errorf("provider %s: %s: %s", p.name, resp.Status, string(respBody))
	}

	var result googleResponse
	if err := json.Unmarshal(respBody, &result); err != nil {
		return nil, fmt.Errorf("provider %s: %w", p.name, err)
	}

	if len(result.Candidates) == 0 || len(result.Candidates[0].Content.Parts) == 0 {
		return nil, fmt.Errorf("provider %s: empty response", p.name)
	}

	return &llm.CompletionResponse{
		Content:      result.Candidates[0].Content.Parts[0].Text,
		InputTokens:  result.UsageMetadata.PromptTokenCount,
		OutputTokens: result.UsageMetadata.CandidatesTokenCount,
		Model:        p.model,
		Provider:     p.name,
	}, nil
}

func (p *GoogleProvider) Stream(ctx context.Context, req llm.CompletionRequest) (<-chan llm.Token, error) {
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
