package tui

import (
	"os"
	"testing"
)

// TestMain isolates tests from the developer's per-user BOI directory, which
// holds a shared .env and cached Provider qualification profiles.
func TestMain(m *testing.M) {
	dir, err := os.MkdirTemp("", "boi-config-test-*")
	if err != nil {
		panic(err)
	}
	os.Setenv("BOI_CONFIG_DIR", dir)
	code := m.Run()
	os.RemoveAll(dir)
	os.Exit(code)
}
