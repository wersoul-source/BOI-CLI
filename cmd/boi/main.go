package main

import (
	"fmt"
	"os"
	"path/filepath"

	"github.com/boi-family/boi-cli/internal/cli"
	"github.com/boi-family/boi-cli/internal/config"
	"github.com/boi-family/boi-cli/internal/envfile"
	"github.com/boi-family/boi-cli/internal/term"
	"github.com/boi-family/boi-cli/internal/workspace"
)

func main() {
	// Fix Thai/UTF-8 rendering in every terminal (WT, cmd, mintty, wezterm...)
	// Must be called before any fmt.Println or terminal I/O
	term.SetUTF8Console()

	if len(os.Args) <= 1 {
		firstRunExperience()
		runTUI()
		return
	}

	// Load .env for CLI commands (boi ask, boi config, etc.)
	root := detectRoot()
	envfile.Load(filepath.Join(root, ".env"))

	if err := cli.Execute(); err != nil {
		fmt.Fprintf(os.Stderr, "Error: %v\n", err)
		os.Exit(1)
	}
}

func firstRunExperience() {
	root := detectRoot()
	boiDir := filepath.Join(root, ".boi")
	envPath := filepath.Join(root, ".env")

	envfile.Load(envPath)

	needsInit := false
	if _, err := os.Stat(boiDir); os.IsNotExist(err) {
		needsInit = true
	} else {
		if _, err := os.Stat(filepath.Join(boiDir, "config.yaml")); os.IsNotExist(err) {
			needsInit = true
		}
	}

	if needsInit {
		fmt.Println()
		fmt.Println("🔧 First run — initializing BOI CLI workspace...")
		if err := cli.RunInit(true); err != nil {
			fmt.Printf("  Warning: auto-init failed: %v\n", err)
		} else {
			fmt.Println("  ✅ Workspace initialized.")
		}

		// Ensure Thai font available (user-level install, no admin required)
		// On Windows 8+ Leelawadee UI already exists → this is a fast no-op
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
			cli.RunSetupWizard(false)
			envfile.Load(envPath)
			fmt.Println()
		}

		if needsPersona {
			fmt.Println("  Step 2: Choose Your Persona")
			cli.RunPersonaWizard()
			fmt.Println()
		}
	}

	fmt.Println()
	fmt.Println("🚀 Launching BOI CLI...")
	fmt.Println()
}

func detectRoot() string {
	root, err := workspace.DetectRoot()
	if err != nil {
		root, _ = os.Getwd()
	}
	return root
}
