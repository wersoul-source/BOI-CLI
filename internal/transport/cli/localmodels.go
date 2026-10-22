package cli

import (
	"context"
	"encoding/json"
	"io"
	"net/http"
	"sort"
	"strings"
	"time"
)

// localServer is a model server found on this machine.
type localServer struct {
	Label   string
	BaseURL string
	Models  []string
}

type localCandidate struct{ Label, BaseURL string }

// localCandidates are where the common local servers listen by default. All
// of them speak the OpenAI-compatible API, so BOI uses them as "ollama".
var localCandidates = []localCandidate{
	{"Ollama", "http://localhost:11434/v1"},
	{"LM Studio", "http://localhost:1234/v1"},
	{"llama.cpp", "http://localhost:8080/v1"},
}

// detectLocalServers asks each candidate for its model list and returns the
// ones that answered with at least one chat model.
func detectLocalServers(ctx context.Context, client *http.Client, candidates []localCandidate) []localServer {
	var found []localServer
	for _, candidate := range candidates {
		models := listModels(ctx, client, candidate.BaseURL)
		if len(models) > 0 {
			found = append(found, localServer{Label: candidate.Label, BaseURL: candidate.BaseURL, Models: models})
		}
	}
	return found
}

func listModels(ctx context.Context, client *http.Client, baseURL string) []string {
	ctx, cancel := context.WithTimeout(ctx, 1500*time.Millisecond)
	defer cancel()
	request, err := http.NewRequestWithContext(ctx, http.MethodGet, strings.TrimRight(baseURL, "/")+"/models", nil)
	if err != nil {
		return nil
	}
	response, err := client.Do(request)
	if err != nil {
		return nil
	}
	defer response.Body.Close()
	if response.StatusCode != http.StatusOK {
		return nil
	}
	var body struct {
		Data []struct {
			ID string `json:"id"`
		} `json:"data"`
	}
	if err := json.NewDecoder(io.LimitReader(response.Body, 1<<20)).Decode(&body); err != nil {
		return nil
	}
	var models []string
	for _, item := range body.Data {
		id := strings.TrimSpace(item.ID)
		// Embedding models cannot chat.
		if id == "" || strings.Contains(strings.ToLower(id), "embed") {
			continue
		}
		models = append(models, id)
	}
	sort.Strings(models)
	return models
}
