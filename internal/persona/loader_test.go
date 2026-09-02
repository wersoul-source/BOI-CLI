package persona

import "testing"

func TestCorePersonaIsBoi(t *testing.T) {
	t.Parallel()

	p := CorePersona()
	if p == nil {
		t.Fatal("CorePersona returned nil")
	}
	if p.Name != "boi" {
		t.Fatalf("Core Persona name = %q, want boi", p.Name)
	}
	if p.SystemPrompt == "" {
		t.Fatal("Core Persona system prompt is empty")
	}
}
