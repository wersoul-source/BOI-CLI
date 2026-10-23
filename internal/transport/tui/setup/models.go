package setup

import (
	"context"
	"encoding/json"
	"io"
	"net/http"
	"sort"
	"strings"
	"time"
)

// fetchModels asks an OpenAI-compatible server which models the key may use,
// so the list is never older than the provider. It returns nil on any failure
// and the wizard falls back to the built-in catalog. A test replaces it.
var fetchModels = func(baseURL, apiKey string) []string {
	ctx, cancel := context.WithTimeout(context.Background(), 4*time.Second)
	defer cancel()
	request, err := http.NewRequestWithContext(ctx, http.MethodGet, strings.TrimRight(baseURL, "/")+"/models", nil)
	if err != nil {
		return nil
	}
	if apiKey != "" {
		request.Header.Set("Authorization", "Bearer "+apiKey)
	}
	response, err := http.DefaultClient.Do(request)
	if err != nil {
		return nil
	}
	defer response.Body.Close()
	if response.StatusCode != http.StatusOK {
		return nil
	}
	return parseModelIDs(io.LimitReader(response.Body, 2<<20))
}

// parseModelIDs reads an OpenAI-style {"data":[{"id":...}]} list and drops
// models that cannot chat.
func parseModelIDs(r io.Reader) []string {
	var body struct {
		Data []struct {
			ID string `json:"id"`
		} `json:"data"`
	}
	if err := json.NewDecoder(r).Decode(&body); err != nil {
		return nil
	}
	var models []string
	for _, item := range body.Data {
		id := strings.TrimSpace(item.ID)
		lower := strings.ToLower(id)
		if id == "" || strings.Contains(lower, "embed") || strings.Contains(lower, "moderation") || strings.Contains(lower, "whisper") || strings.Contains(lower, "tts") {
			continue
		}
		models = append(models, id)
	}
	sort.Strings(models)
	return models
}

// liveModels returns the provider's own model list when it speaks the
// OpenAI-compatible API; Anthropic and Google use other protocols.
func liveModels(name, baseURL, apiKey string) []string {
	switch name {
	case "anthropic", "google":
		return nil
	}
	if strings.TrimSpace(baseURL) == "" {
		return nil
	}
	return fetchModels(baseURL, apiKey)
}
