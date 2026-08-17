package cli

import (
	"fmt"
	"os"
	"path/filepath"

	"github.com/boi-family/boi-cli/internal/app"
	"github.com/boi-family/boi-cli/internal/config"
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

	needsSetup := !envfile.HasProviders(envPath)
	needsPersona := true

	cfgPath := filepath.Join(boiDir, "config.yaml")
	cfg, err := config.LoadFrom(cfgPath)
	if err == nil && cfg.Persona != "" {
		needsPersona = false
	}

	if needsSetup || needsPersona {
		fmt.Println()
		fmt.Println("⚡ Quick Setup — let's get BOI ready:")
		fmt.Println()

		if needsSetup {
			fmt.Println("  Step 1: Configure AI Providers")
			fmt.Println("  (TUI wizard — arrow keys, model picker)")
			fmt.Println()
			RunSetupWizard(false)
			envfile.Load(envPath)
			fmt.Println()
		}

		if needsPersona {
			fmt.Println("  Step 2: Choose Your Persona")
			RunPersonaWizard()
			fmt.Println()
		}
	}

	fmt.Println()
	fmt.Println("🚀 Launching BOI CLI...")
	fmt.Println()
}
