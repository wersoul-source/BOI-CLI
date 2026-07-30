package config

import (
	"fmt"
	"os"
	"path/filepath"

	"gopkg.in/yaml.v3"
)

// Config holds all BOI CLI configuration
type Config struct {
	// Provider is the default LLM provider
	Provider string `yaml:"provider"`

	// Model is the default model to use
	Model string `yaml:"model"`

	// Persona is the active BOI Family persona name
	Persona string `yaml:"persona"`

	// LogLevel controls logging verbosity (debug, info, warn, error)
	LogLevel string `yaml:"log_level"`

	// Workspace is the project root (auto-detected if empty)
	Workspace string `yaml:"workspace"`

	// APIKeys holds provider API keys (never committed)
	APIKeys map[string]string `yaml:"api_keys"`

	// Sandbox settings
	Sandbox SandboxConfig `yaml:"sandbox"`
}

// SandboxConfig controls command sandbox behavior
type SandboxConfig struct {
	// Enabled enables sandboxing (default: true)
	Enabled bool `yaml:"enabled"`

	// DenyCommands is a list of blocked patterns
	DenyCommands []string `yaml:"deny_commands"`
}

// LoadFrom loads configuration from a YAML file
func LoadFrom(path string) (*Config, error) {
	data, err := os.ReadFile(path)
	if err != nil {
		return nil, fmt.Errorf("read config: %w", err)
	}

	cfg := Default()
	if err := yaml.Unmarshal(data, cfg); err != nil {
		return nil, fmt.Errorf("parse config: %w", err)
	}

	// Apply env var overrides
	cfg.applyEnvOverrides()

	return cfg, nil
}

// SaveTo writes configuration to a YAML file
func (c *Config) SaveTo(path string) error {
	dir := filepath.Dir(path)
	if err := os.MkdirAll(dir, 0755); err != nil {
		return fmt.Errorf("create config dir: %w", err)
	}

	data, err := yaml.Marshal(c)
	if err != nil {
		return fmt.Errorf("marshal config: %w", err)
	}

	return os.WriteFile(path, data, 0644)
}

// applyEnvOverrides applies environment variable overrides
func (c *Config) applyEnvOverrides() {
	if v := os.Getenv("BOI_PROVIDER"); v != "" {
		c.Provider = v
	}
	if v := os.Getenv("BOI_MODEL"); v != "" {
		c.Model = v
	}
	if v := os.Getenv("BOI_LOG_LEVEL"); v != "" {
		c.LogLevel = v
	}

	// Load API keys from env
	if c.APIKeys == nil {
		c.APIKeys = make(map[string]string)
	}
	envKeys := map[string]string{
		"openai":    "OPENAI_API_KEY",
		"anthropic": "ANTHROPIC_API_KEY",
		"google":    "GOOGLE_API_KEY",
		"groq":      "GROQ_API_KEY",
		"deepseek":  "DEEPSEEK_API_KEY",
	}
	for provider, env := range envKeys {
		if v := os.Getenv(env); v != "" {
			c.APIKeys[provider] = maskKey(v)
		}
	}
}

// maskKey masks an API key showing only last 4 chars
func maskKey(key string) string {
	if len(key) <= 8 {
		return "***"
	}
	return key[:4] + "..." + key[len(key)-4:]
}

func (c *Config) String() string {
	return fmt.Sprintf("Provider=%s Model=%s LogLevel=%s", c.Provider, c.Model, c.LogLevel)
}
