package tui

import (
	"fmt"
	"strings"

	term "github.com/boi-family/boi-cli/internal/platform/terminal"
)

var spinnerChars = []string{"⠋", "⠙", "⠹", "⠸", "⠼", "⠴", "⠦", "⠧", "⠇", "⠏"}

type StatusModel struct {
	level     string
	agentName string
	provider  string
	status    string
	usagePct  int
	spinIdx   int
	width     int
}

func NewStatus(agentName, provider string, _ []string) StatusModel {
	if provider == "" {
		provider = "none"
	}

	return StatusModel{
		level:     "Mid",
		agentName: agentName,
		provider:  provider,
		status:    "idle",
	}
}

func (s *StatusModel) SetStatus(st string) {
	s.status = st
}

func (s *StatusModel) Status() string {
	return s.status
}

func (s *StatusModel) SetAgentName(name string) {
	s.agentName = name
}

func (s *StatusModel) SetProvider(provider string) {
	s.provider = provider
}

func (s *StatusModel) SetUsagePct(pct int) {
	s.usagePct = pct
}

func (s *StatusModel) SetWidth(w int) {
	s.width = w
}

// Height returns the fixed status bar height (1 row).
func (s *StatusModel) Height() int {
	return 1
}

func (s *StatusModel) AgentName() string {
	return s.agentName
}

func (s *StatusModel) Tick() {
	if s.status == "thinking" || s.status == "working" || s.status == "cancelling" {
		s.spinIdx = (s.spinIdx + 1) % len(spinnerChars)
	}
}

func (s *StatusModel) View() string {
	statusIcon := "○"
	switch s.status {
	case "thinking", "working", "cancelling":
		statusIcon = spinnerChars[s.spinIdx]
	case "approval":
		statusIcon = "!"
	case "idle":
		statusIcon = "✓"
	case "error":
		statusIcon = "✗"
	}

	left := fmt.Sprintf(" BOI CLI  [%s] ", s.level)
	mid := fmt.Sprintf(" Agent: %s ", s.agentName)

	// Provider usage bar
	provStr := s.provider
	if s.usagePct > 0 && s.usagePct < 100 {
		bar := usageBar(s.usagePct)
		provStr = fmt.Sprintf("%s %s", s.provider, bar)
	}
	right := fmt.Sprintf(" %s: %s  %s ", provStr, s.status, statusIcon)

	available := s.width - term.ThaiStringWidth(left) - term.ThaiStringWidth(mid) - term.ThaiStringWidth(right)
	if available < 0 {
		available = 0
	}

	padding := strings.Repeat(" ", available)

	return StatusBarStyle.Width(s.width).Render(left + mid + padding + right)
}

// usageBar returns a compact 6-char bar representing percentage.
// "████░░" = ~67%, "█████░" = ~83%, "██░░░░" = ~33%
func usageBar(pct int) string {
	if pct >= 100 {
		return "██████"
	}
	filled := (pct + 8) / 17 // 6 segments, each ~16.7%
	if filled < 0 {
		filled = 0
	}
	if filled > 6 {
		filled = 6
	}
	return strings.Repeat("█", filled) + strings.Repeat("░", 6-filled)
}
