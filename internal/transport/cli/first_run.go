package cli

import (
	"bufio"
	"context"
	"fmt"
	"io"
	"os"
	"path/filepath"
	"strings"
	"time"

	"github.com/boi-family/boi-cli/internal/app"
	term "github.com/boi-family/boi-cli/internal/platform/terminal"
	"github.com/boi-family/boi-cli/internal/service/config/envfile"
	llmfactory "github.com/boi-family/boi-cli/internal/service/provider/factory"
)

func loadEnvFiles(workspaceEnv string) {
	envfile.Load(workspaceEnv)
	if global := app.GlobalEnvPath(); global != "" {
		envfile.Load(global)
	}
}

// offerQualification closes the most common first-run gap: a Provider is
// configured but has never been qualified, so the TUI would start with no
// usable Agent. Qualification sends real requests, so the user is asked.
func offerQualification(runtime *app.Runtime, in io.Reader, out io.Writer) {
	configured, err := llmfactory.LoadConfiguredProvidersFromEnv()
	if err != nil || len(configured) == 0 || len(app.QualifiedProviders(runtime.BoiDir, configured)) > 0 {
		return
	}
	item := configured[0]
	fmt.Fprintf(out, "\n🔎 Provider %s (%s) is configured but not qualified yet.\n", item.Name, item.Model)
	fmt.Fprint(out, "   Run the qualification probes now? This sends a few small paid requests. [Y/n] ")
	answer, _ := bufio.NewReader(in).ReadString('\n')
	if answer = strings.ToLower(strings.TrimSpace(answer)); answer != "" && answer != "y" && answer != "yes" {
		fmt.Fprintf(out, "   Skipped. Run 'boi provider qualify %s' before asking the Agent.\n", item.Name)
		return
	}
	if err := qualifyProvider(context.Background(), runtime, item, out, 30*time.Second); err != nil {
		fmt.Fprintf(out, "   Qualification failed: %v\n", err)
	}
}

// RunFirstRun prepares a workspace before the interactive TUI starts.
func RunFirstRun(runtime *app.Runtime) {
	boiDir := runtime.BoiDir
	envPath := runtime.EnvPath

	loadEnvFiles(envPath)

	needsInit := false
	if _, err := os.Stat(boiDir); os.IsNotExist(err) {
		needsInit = true
	} else if _, err := os.Stat(filepath.Join(boiDir, "config.yaml")); os.IsNotExist(err) {
		needsInit = true
	}

	if needsInit {
		fmt.Println()
		fmt.Println("🔧 First run — initializing BOI CLI workspace...")
		if err := RunInitAt(runtime.WorkspaceRoot, true); err != nil {
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
	if err := runtime.EnsureWorkspaceState(); err != nil {
		fmt.Printf("  Warning: workspace state unavailable: %v\n", err)
	}

	// Providers may come from the workspace .env, the per-user .env, or the
	// shell environment; only ask for setup when none of them has one.
	needsSetup := llmfactory.CountProvidersFromEnv() == 0

	if needsSetup {
		fmt.Println()
		fmt.Println("⚡ Quick Setup — let's get BOI ready:")
		fmt.Println()
		fmt.Println("  Step 1: Configure AI Providers")
		fmt.Println("  (TUI wizard — arrow keys, model picker)")
		fmt.Println()
		if err := RunSetupWizard(false); err != nil {
			fmt.Printf("  Warning: provider setup failed: %v\n", err)
		}
		loadEnvFiles(envPath)
		fmt.Println()
	}
	offerQualification(runtime, os.Stdin, os.Stdout)

	if _, created, err := EnsureAgentIdentity(runtime, os.Stdin, os.Stdout); err != nil {
		fmt.Printf("  Warning: Agent identity is unavailable: %v\n", err)
	} else if created {
		fmt.Println()
	}

	fmt.Println()
	fmt.Println("🚀 Launching BOI CLI...")
	fmt.Println()
}
