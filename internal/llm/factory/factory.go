package factory

import (
	"fmt"
	"os"

	"github.com/boi-family/boi-cli/internal/llm"
	"github.com/boi-family/boi-cli/internal/llm/providers"
)

type ProviderConfig struct {
	Name    string
	APIKey  string
	BaseURL string
	Model   string
}

func LoadProvidersFromEnv() ([]llm.Provider, error) {
	var providers []llm.Provider
	for i := 1; i <= 4; i++ {
		prefix := fmt.Sprintf("PSC_%d_", i)
		cfg := ProviderConfig{
			Name:    os.Getenv(prefix + "NAME"),
			APIKey:  os.Getenv(prefix + "API_KEY"),
			BaseURL: os.Getenv(prefix + "BASE_URL"),
			Model:   os.Getenv(prefix + "MODEL"),
		}
		if cfg.Name == "" || cfg.APIKey == "" || cfg.Model == "" {
			continue
		}
		p := createProvider(cfg)
		if p != nil {
			providers = append(providers, p)
		}
	}
	if len(providers) == 0 {
		return nil, fmt.Errorf("no valid providers found in environment (PSC_1..PSC_4)")
	}
	return providers, nil
}

func createProvider(cfg ProviderConfig) llm.Provider {
	switch cfg.Name {
	case "openai":
		return providers.NewOpenAIProvider(cfg.Name, cfg.APIKey, cfg.BaseURL, cfg.Model)
	case "anthropic":
		return providers.NewAnthropicProvider(cfg.Name, cfg.APIKey, cfg.BaseURL, cfg.Model)
	default:
		return nil
	}
}
