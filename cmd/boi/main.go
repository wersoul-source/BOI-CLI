package main

import (
	"fmt"
	"os"

	"github.com/boi-family/boi-cli/internal/app"
	term "github.com/boi-family/boi-cli/internal/platform/terminal"
	"github.com/boi-family/boi-cli/internal/service/config/envfile"
	"github.com/boi-family/boi-cli/internal/transport/cli"
)

func main() {
	// Fix Thai/UTF-8 rendering in every terminal (WT, cmd, mintty, wezterm...)
	// Must be called before any fmt.Println or terminal I/O
	term.SetUTF8Console()
	runtime, err := app.NewRuntime(cli.Version)
	if err != nil {
		fmt.Fprintf(os.Stderr, "Error: initialize runtime: %v\n", err)
		os.Exit(1)
	}

	if len(os.Args) <= 1 {
		cli.RunFirstRun(runtime)
		runTUI(runtime)
		return
	}

	// Load .env for CLI commands (boi ask, boi config, etc.): workspace
	// first, then the per-user file; an earlier value is never overridden.
	envfile.Load(runtime.EnvPath)
	if global := app.GlobalEnvPath(); global != "" {
		envfile.Load(global)
	}

	if err := cli.Execute(runtime); err != nil {
		if !cli.ErrorReported(err) {
			fmt.Fprintf(os.Stderr, "Error: %v\n", err)
		}
		os.Exit(cli.ExitCode(err))
	}
}
