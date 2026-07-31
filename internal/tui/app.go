package tui

import (
	"fmt"
	"path/filepath"
	"strings"
	"time"

	tea "github.com/charmbracelet/bubbletea"
	"github.com/charmbracelet/lipgloss"
	"github.com/boi-family/boi-cli/internal/config"
	"github.com/boi-family/boi-cli/internal/persona"
	"github.com/boi-family/boi-cli/internal/workspace"
)

type tickMsg time.Time

type Model struct {
	splash *SplashModel
	chat   ChatModel
	input  InputModel
	status StatusModel
	help   HelpModel
	width  int
	height int
	mode   string
}

func NewApp() *Model {
	var personaNames []string
	activePersona := "kamkaew"
	provider := "none"
	root := ""

	r, err := workspace.DetectRoot()
	if err == nil {
		root = r
		personaDir := filepath.Join(workspace.GetBoiDir(root), "personas")
		reg, regErr := persona.Load(personaDir)
		if regErr == nil {
			personaNames = reg.List()
			cfgPath := filepath.Join(workspace.GetBoiDir(root), "config.yaml")
			cfg, cfgErr := config.LoadFrom(cfgPath)
			if cfgErr == nil && cfg.Persona != "" {
				activePersona = cfg.Persona
			}
			if cfgErr == nil && cfg.Provider != "" {
				if cfg.Model != "" {
					provider = fmt.Sprintf("%s/%s", cfg.Provider, cfg.Model)
				} else {
					provider = cfg.Provider
				}
			}
		}
	}

	if len(personaNames) == 0 {
		personaNames = []string{"kamkaew", "kampun", "dang", "don", "kine", "boi"}
	}

	boiDir := filepath.Join(root, ".boi")
	memoryCount := countMemoryEntries(filepath.Join(boiDir, "memory"))
	skillList, skillCount := listSkills(filepath.Join(boiDir, "skills"))
	providerCount := countPSCProviders()

	splash := NewSplash(root, len(personaNames), strings.Join(personaNames, ", "), providerCount, memoryCount, skillCount, strings.Join(skillList, ", "))

	m := &Model{
		splash: splash,
		chat:   NewChat(),
		input:  NewInput(),
		status: NewStatus(activePersona, provider, personaNames),
		help:   NewHelp(),
		mode:   "splash",
	}

	return m
}

func (m *Model) Init() tea.Cmd {
	return tea.Batch(
		tea.EnterAltScreen,
		splashTimeoutCmd(),
	)
}

func tickCmd() tea.Cmd {
	return tea.Tick(time.Millisecond*100, func(t time.Time) tea.Msg {
		return tickMsg(t)
	})
}

func (m *Model) transitionToChat() tea.Cmd {
	m.mode = "chat"
	m.status.SetWidth(m.width)
	m.help.SetWidth(m.width)
	m.chat.SetSize(m.width, m.height-6)
	m.input.SetWidth(m.width)
	m.chat.AddMessage("system", "BOI CLI ready. Type /help for commands.")
	return tickCmd()
}

func (m *Model) Update(msg tea.Msg) (tea.Model, tea.Cmd) {
	if m.mode == "splash" {
		switch msg := msg.(type) {
		case tea.WindowSizeMsg:
			m.width = msg.Width
			m.height = msg.Height
			m.splash.SetSize(msg.Width, msg.Height)
			return m, nil

		case tea.KeyMsg:
			switch msg.String() {
			case "ctrl+c", "esc":
				return m, tea.Quit
			case "enter":
				return m, m.transitionToChat()
			}
			return m, nil

		case splashTimeoutMsg:
			return m, m.transitionToChat()
		}
		return m, nil
	}

	var cmds []tea.Cmd

	switch msg := msg.(type) {
	case tea.WindowSizeMsg:
		m.width = msg.Width
		m.height = msg.Height
		m.status.SetWidth(msg.Width)
		m.help.SetWidth(msg.Width)
		m.chat.SetSize(msg.Width, msg.Height-6)
		m.input.SetWidth(msg.Width)

	case tickMsg:
		m.status.Tick()
		cmds = append(cmds, tickCmd())

	case tea.KeyMsg:
		switch msg.String() {
		case "ctrl+c", "ctrl+q", "esc":
			return m, tea.Quit

		case "enter":
			val := strings.TrimSpace(m.input.Value())
			if val != "" {
				m.chat.AddMessage("user", val)
				m.input.Reset()
				m.status.SetStatus("thinking")
				go m.processInput(val)
			}
			return m, tea.Batch(cmds...)

		case "tab":
			newPersona := m.status.SwitchPersona()
			m.chat.AddMessage("system", fmt.Sprintf("Switched to persona: %s", newPersona))
			return m, nil

		case "ctrl+l":
			m.chat.Clear()
			m.chat.AddMessage("system", "Chat cleared")
			return m, tea.Batch(cmds...)

		case "/":
			m.chat.AddMessage("system", "Commands: /help /persona /clear /quit /config")
			return m, nil
		}
	}

	var cmd tea.Cmd
	m.chat, cmd = m.chat.Update(msg)
	if cmd != nil {
		cmds = append(cmds, cmd)
	}
	m.input, cmd = m.input.Update(msg)
	if cmd != nil {
		cmds = append(cmds, cmd)
	}

	return m, tea.Batch(cmds...)
}

func (m *Model) processInput(input string) {
	time.Sleep(300 * time.Millisecond)

	lower := strings.ToLower(strings.TrimSpace(input))

	switch {
	case lower == "/help":
		m.chat.AddMessage("system", helpText)
	case lower == "/persona":
		names := m.status.Personas()
		m.chat.AddMessage("system", fmt.Sprintf("Personas: %s\nTab to cycle", strings.Join(names, ", ")))
	case lower == "/clear":
		m.chat.Clear()
		m.chat.AddMessage("system", "Chat cleared")
	case lower == "/quit":
		m.chat.AddMessage("system", "Use Ctrl+Q or Esc to quit")
	case strings.HasPrefix(lower, "/"):
		m.chat.AddMessage("agent", fmt.Sprintf("Command not recognized: %s\nType /help for available commands.", input))
	default:
		m.chat.AddMessage("agent", fmt.Sprintf("Received: %s\n\nThis is the MVP echo response. The agent will process your message in future versions.", input))
	}

	m.status.SetStatus("idle")
}

func (m *Model) View() string {
	if m.mode == "splash" {
		return m.splash.View()
	}

	status := m.status.View()
	chat := m.chat.View()
	input := m.input.View()
	help := m.help.View()

	return lipgloss.JoinVertical(lipgloss.Top,
		status,
		chat,
		input,
		help,
	)
}

var helpText = `Available commands:
  /help     Show this help
  /persona  List available personas
  /clear    Clear chat history
  /quit     Exit BOI CLI
  /config   Show current configuration

Keyboard shortcuts:
  Enter     Send message
  Tab       Switch persona
  Ctrl+N    Newline in input
  Ctrl+L    Clear chat
  Ctrl+Q    Quit
  Esc       Quit
  /         Quick command`
