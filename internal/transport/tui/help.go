package tui

import (
	"fmt"
	"strings"
)

var availableCommands = []string{
	"/help",
	"/persona",
	"/clear",
	"/quit",
	"/config",
	"/provider",
	"/model",
	"/workspace",
	"/ls",
	"/read",
}

type HelpModel struct {
	width       int
	show        bool
	matchPrefix string // current input prefix for suggestion matching
}

func NewHelp() HelpModel {
	return HelpModel{show: true}
}

func (h *HelpModel) SetWidth(w int) {
	h.width = w
}

// Height returns the current help bar height (0 when hidden).
func (h *HelpModel) Height() int {
	if h.show {
		return 1
	}
	return 0
}

func (h *HelpModel) Toggle() {
	h.show = !h.show
}

// SetSuggestions updates the match prefix for command suggestions
func (h *HelpModel) SetSuggestions(prefix string) {
	h.matchPrefix = prefix
}

// BestMatch returns the first command that starts with the given prefix (for autocomplete)
func (h *HelpModel) BestMatch(prefix string) string {
	lower := strings.ToLower(prefix)
	for _, cmd := range availableCommands {
		if strings.HasPrefix(cmd, lower) {
			return cmd
		}
	}
	return ""
}

func (h *HelpModel) View() string {
	if !h.show {
		return ""
	}

	var content string

	if strings.HasPrefix(h.matchPrefix, "/") {
		// Command suggestion mode
		matched := matchCommands(h.matchPrefix)
		if len(matched) > 0 {
			parts := make([]string, 0, len(matched)+1)
			parts = append(parts, "▸")
			for _, cmd := range matched {
				if cmd == h.BestMatch(h.matchPrefix) && h.matchPrefix != cmd {
					parts = append(parts, HighlightStyle.Render(cmd))
				} else {
					parts = append(parts, SuggestionStyle.Render(cmd))
				}
			}
			content = strings.Join(parts, " ")
		} else {
			content = "▸ " + DimStyle.Render("unknown command — type /help for available commands")
		}
	} else {
		// Normal help bar
		content = "Enter:send  Tab:persona  Ctrl+Q:quit  /:commands  Ctrl+L:clear"
	}

	if h.width > 0 {
		return HelpStyle.Width(h.width).Render(content)
	}
	return HelpStyle.Render(content)
}

func matchCommands(prefix string) []string {
	lower := strings.ToLower(prefix)
	var result []string
	for _, cmd := range availableCommands {
		if strings.HasPrefix(cmd, lower) {
			result = append(result, cmd)
		}
	}
	if len(result) == 0 {
		// Show all as hint
		result = availableCommands
	}
	return result
}

// For external use: returns a formatted command list
func CommandList() string {
	return fmt.Sprintf("▸ %s", strings.Join(availableCommands, "  "))
}
