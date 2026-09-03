package cli

import (
	"fmt"
	"os"
	"path/filepath"

	"github.com/boi-family/boi-cli/internal/app"
	"github.com/boi-family/boi-cli/internal/config/envfile"
	term "github.com/boi-family/boi-cli/internal/platform/terminal"
)

// RunFirstRun prepares a workspace before the interactive TUI starts.
func RunFirstRun(runtime *app.Runtime) {
	boiDir := runtime.BoiDir
	envPath := runtime.EnvPath

	envfile.Load(envPath)

	needsInit := false
	if _, err := os.Stat(boiDir); os.IsNotExist(err) {
		needsInit = true
	} else if _, err := os.Stat(filepath.Join(boiDir, "config.yaml")); os.IsNotExist(err) {
		needsInit = true
	}

	if needsInit {
		fmt.Println()
		fmt.Println("🔧 First run — initializing BOI CLI workspace...")
		if err := RunInit(true); err != nil {
			fmt.Printf("  Warning: auto-init failed: %v\n", err)
		} else {
			fmt.Println("  ✅ Workspace initialized.")
		}

		installed, err := term.EnsureThaiFont()
		if err != nil {
			fmt.Printf("  ⚠️  Thai font check failed: %v\n", err)
		} else if installed {
			fmt.Println("  ✅ Thai font installed (Noto Sans Thai)")
		}
	}
	if err := app.EnsureCapabilityIndexes(boiDir); err != nil {
		fmt.Printf("  Warning: capability indexes unavailable: %v\n", err)
	}

	needsSetup := !envfile.HasProviders(envPath)

	if needsSetup {
		fmt.Println()
		fmt.Println("⚡ Quick Setup — let's get BOI ready:")
		fmt.Println()
		fmt.Println("  Step 1: Configure AI Providers")
		fmt.Println("  (TUI wizard — arrow keys, model picker)")
		fmt.Println()
		RunSetupWizard(false)
		envfile.Load(envPath)
		fmt.Println()
	}

	if _, created, err := EnsureAgentIdentity(runtime, os.Stdin, os.Stdout); err != nil {
		fmt.Printf("  Warning: Agent identity is unavailable: %v\n", err)
	} else if created {
		fmt.Println()
	}

	fmt.Println()
	fmt.Println("🚀 Launching BOI CLI...")
	fmt.Println()
}
