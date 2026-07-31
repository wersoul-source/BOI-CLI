package tui

import (
	"fmt"
	"strings"

	"github.com/charmbracelet/lipgloss"
)

var spinnerChars = []string{"⠋", "⠙", "⠹", "⠸", "⠼", "⠴", "⠦", "⠧", "⠇", "⠏"}

type StatusModel struct {
	level      string
	persona    string
	provider   string
	status     string
	spinIdx    int
	personas   []string
	personaIdx int
	width      int
}

func NewStatus(personaName, provider string, personaNames []string) StatusModel {
	idx := 0
	for i, name := range personaNames {
		if name == personaName {
			idx = i
			break
		}
	}

	if provider == "" {
		provider = "none"
	}

	return StatusModel{
		level:      "Mid",
		persona:    personaName,
		provider:   provider,
		status:     "idle",
		personas:   personaNames,
		personaIdx: idx,
	}
}

func (s *StatusModel) SetStatus(st string) {
	s.status = st
}

func (s *StatusModel) Status() string {
	return s.status
}

func (s *StatusModel) SetPersona(name string) {
	s.persona = name
}

func (s *StatusModel) SetProvider(provider string) {
	s.provider = provider
}

func (s *StatusModel) SetWidth(w int) {
	s.width = w
}

func (s *StatusModel) SwitchPersona() string {
	if len(s.personas) == 0 {
		return s.persona
	}
	s.personaIdx = (s.personaIdx + 1) % len(s.personas)
	s.persona = s.personas[s.personaIdx]
	return s.persona
}

func (s *StatusModel) CurrentPersona() string {
	return s.persona
}

func (s *StatusModel) Personas() []string {
	return s.personas
}

func (s *StatusModel) Tick() {
	if s.status == "thinking" {
		s.spinIdx = (s.spinIdx + 1) % len(spinnerChars)
	}
}

func (s *StatusModel) View() string {
	statusIcon := "○"
	switch s.status {
	case "thinking":
		statusIcon = spinnerChars[s.spinIdx]
	case "idle":
		statusIcon = "✓"
	case "error":
		statusIcon = "✗"
	}

	left := fmt.Sprintf(" BOI CLI  [%s] ", s.level)
	mid := fmt.Sprintf(" Persona: %s ", s.persona)
	right := fmt.Sprintf(" %s: %s  %s ", s.provider, s.status, statusIcon)

	available := s.width - lipgloss.Width(left) - lipgloss.Width(mid) - lipgloss.Width(right)
	if available < 0 {
		available = 0
	}

	padding := strings.Repeat(" ", available)

	return StatusBarStyle.Width(s.width).Render(left + mid + padding + right)
}
