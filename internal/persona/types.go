package persona

type Persona struct {
	Name              string   `yaml:"name"`
	Description       string   `yaml:"description"`
	Model             string   `yaml:"model"`
	Temperature       float64  `yaml:"temperature"`
	PreferredProviders []string `yaml:"preferred_providers"`
	SystemPrompt      string   `yaml:"system_prompt"`
	MaxTokens         int      `yaml:"max_tokens"`
}
