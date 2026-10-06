package config

// Default returns a Config with sensible defaults
func Default() *Config {
	return &Config{
		Provider: "openai",
		Model:    "gpt-4.1-mini",
		Persona:  "boi",
		LogLevel: "info",
		APIKeys:  make(map[string]string),
		Sandbox: SandboxConfig{
			Enabled: true,
			DenyCommands: []string{
				"sudo ",
				"mkfs.",
				"dd if=",
			},
		},
	}
}
