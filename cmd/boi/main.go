package main

import (
	"bufio"
	"fmt"
	"os"
	"path/filepath"
	"strings"

	"github.com/boi-family/boi-cli/internal/cli"
	"github.com/boi-family/boi-cli/internal/config"
	"github.com/boi-family/boi-cli/internal/envfile"
	"github.com/boi-family/boi-cli/internal/workspace"
)

func main() {
	if len(os.Args) <= 1 {
		firstRunExperience()
		runTUI()
		return
	}

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
		fmt.Println("🔧 First run detected! Initializing BOI CLI workspace...")
		if err := cli.RunInit(true); err != nil {
			fmt.Printf("  Warning: auto-init failed: %v\n", err)
		} else {
			fmt.Println("  ✅ Workspace initialized.")
		}
		fmt.Println()
	}

	needsSetup := !envfile.HasProviders(envPath)
	needsPersona := true

	cfgPath := filepath.Join(boiDir, "config.yaml")
	cfg, err := config.LoadFrom(cfgPath)
	if err == nil && cfg.Persona != "" {
		needsPersona = false
	}

	reader := bufio.NewReader(os.Stdin)

	if needsSetup {
		fmt.Println("⚠️  No AI providers configured.")
		fmt.Print("   Set up AI providers now? [Y/n]: ")
		answer, _ := reader.ReadString('\n')
		answer = strings.TrimSpace(strings.ToLower(answer))
		if answer == "" || answer == "y" {
			cli.RunSetupWizard()
			envfile.Load(envPath)
		}
		fmt.Println()
	}

	if needsPersona {
		fmt.Println("⚠️  No persona selected.")
		fmt.Print("   Choose a persona now? [Y/n]: ")
		answer, _ := reader.ReadString('\n')
		answer = strings.TrimSpace(strings.ToLower(answer))
		if answer == "" || answer == "y" {
			cli.RunPersonaWizard()
		}
		fmt.Println()
	}

	fmt.Println("Launching BOI CLI...")
	fmt.Println()
}

func detectRoot() string {
	root, err := workspace.DetectRoot()
	if err != nil {
		root, _ = os.Getwd()
	}
	return root
}
