package main

import (
	"fmt"
	"os"

	"github.com/boi-family/boi-cli/internal/cli"
	"github.com/boi-family/boi-cli/internal/tui"
	tea "github.com/charmbracelet/bubbletea"
)

func runTUI() {
	p := tea.NewProgram(
		tui.NewApp(cli.Version),
		tea.WithAltScreen(),
		tea.WithMouseCellMotion(),
	)

	if _, err := p.Run(); err != nil {
		fmt.Fprintf(os.Stderr, "TUI error: %v\n", err)
		os.Exit(1)
	}
}
