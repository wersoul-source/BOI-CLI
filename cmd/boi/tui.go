package main

import (
	"fmt"
	"os"

	"github.com/boi-family/boi-cli/internal/app"
	"github.com/boi-family/boi-cli/internal/transport/tui"
	tea "github.com/charmbracelet/bubbletea"
)

func runTUI(runtime *app.Runtime) {
	p := tea.NewProgram(
		tui.NewApp(runtime),
		tea.WithAltScreen(),
		tea.WithMouseCellMotion(),
	)

	if _, err := p.Run(); err != nil {
		fmt.Fprintf(os.Stderr, "TUI error: %v\n", err)
		os.Exit(1)
	}
}
