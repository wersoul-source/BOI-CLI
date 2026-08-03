package tui

import (
	"context"
	"fmt"
	"path/filepath"
	"strings"
	"time"

	"github.com/boi-family/boi-cli/internal/config"
	"github.com/boi-family/boi-cli/internal/llm"
	llmfactory "github.com/boi-family/boi-cli/internal/llm/factory"
	"github.com/boi-family/boi-cli/internal/persona"
	"github.com/boi-family/boi-cli/internal/workspace"
	tea "github.com/charmbracelet/bubbletea"
	"github.com/charmbracelet/lipgloss"
)

type tickMsg time.Time

type llmResponseMsg struct {
	content  string
	err      error
	provider string
	model    string
	tokens   int
}

type Model struct {
	splash        *SplashModel
	chat          ChatModel
	input         InputModel
	status        StatusModel
	help          HelpModel
	width         int
	height        int
	mode          string
	router        *llm.Router
	activeP       *persona.Persona
	root          string
}

func NewApp(version string) *Model {
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

	splash := NewSplash(root, len(personaNames), strings.Join(personaNames, ", "), providerCount, memoryCount, skillCount, strings.Join(skillList, ", "), version)

	// Load LLM providers and create router
	var router *llm.Router
	llmProviders, llmErr := llmfactory.LoadProvidersFromEnv()
	if llmErr == nil && len(llmProviders) > 0 {
		router = llm.NewRouter(llmProviders)
		// Update status bar provider
		if provider == "none" || provider == "" {
			n := router.ProviderNames()
			if len(n) > 0 {
				provider = n[0]
			}
		}
	}

	// Load active persona for system prompt
	var activeP *persona.Persona
	if root != "" {
		personaDir := filepath.Join(workspace.GetBoiDir(root), "personas")
		if reg, regErr := persona.Load(personaDir); regErr == nil {
			if p, pErr := reg.Get(activePersona); pErr == nil {
				activeP = p
			}
		}
	}

	m := &Model{
		splash:  splash,
		chat:    NewChat(),
		input:   NewInput(),
		status:  NewStatus(activePersona, provider, personaNames),
		help:    NewHelp(),
		mode:    "splash",
		router:  router,
		activeP: activeP,
		root:    root,
	}

	return m
}

func (m *Model) Init() tea.Cmd {
	return tea.EnterAltScreen
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
	m.chat.SetSize(m.width, m.chatHeight())
	m.input.SetWidth(m.width)
	m.chat.AddMessage("system", "BOI CLI ready. Type /help for commands.")
	return tickCmd()
}

// chatHeight computes the chat viewport height from the real layout:
// total - status bar - input box (grows/shrinks) - help bar - 1 breathing row.
func (m *Model) chatHeight() int {
	h := m.height - m.status.Height() - m.input.Height() - m.help.Height() - 1
	if h < 1 {
		h = 1
	}
	return h
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
		m.chat.SetSize(msg.Width, m.chatHeight())
		m.input.SetWidth(msg.Width)

	case tickMsg:
		m.status.Tick()
		cmds = append(cmds, tickCmd())

	case llmResponseMsg:
		m.handleLLMResponse(msg)
		return m, tea.Batch(cmds...)

	case tea.KeyMsg:
		switch msg.String() {
		case "ctrl+c", "ctrl+q", "esc":
			return m, tea.Quit

		case "enter":
			val := strings.TrimSpace(m.input.Value())
			if val != "" {
				isCmd := strings.HasPrefix(val, "/")
				if !isCmd {
					m.chat.AddMessage("user", val)
				}
				m.input.Reset()
				m.help.SetSuggestions("")
				if isCmd {
					// Handle slash commands immediately
					result, clearChat := m.handleSlashCommand(val)
					if clearChat {
						m.chat.Clear()
					}
					if result != "" {
						m.chat.AddMessage("system", result)
					}
					return m, nil
				}
				m.status.SetStatus("thinking")
				cmds = append(cmds, m.callLLMCmd(val))
			}
			return m, tea.Batch(cmds...)

		case "tab":
			// If in command mode, autocomplete instead of switch persona
			if strings.HasPrefix(m.input.Value(), "/") {
				best := m.help.BestMatch(m.input.Value())
				if best != "" && best != m.input.Value() {
					m.input.SetValue(best)
					m.help.SetSuggestions(best)
				}
			} else {
				newPersona := m.status.SwitchPersona()
				m.chat.AddMessage("system", fmt.Sprintf("Switched to persona: %s", newPersona))
				// Reload persona for system prompt
				if m.root != "" {
					personaDir := filepath.Join(workspace.GetBoiDir(m.root), "personas")
					if reg, regErr := persona.Load(personaDir); regErr == nil {
						if p, pErr := reg.Get(newPersona); pErr == nil {
							m.activeP = p
						}
					}
				}
			}
			return m, nil

		case "ctrl+l":
			m.chat.Clear()
			m.chat.AddMessage("system", "Chat cleared")
			return m, tea.Batch(cmds...)
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

	// Update command suggestions based on input prefix
	m.help.SetSuggestions(m.input.Value())

	return m, tea.Batch(cmds...)
}

func (m *Model) handleSlashCommand(input string) (string, bool) {
	lower := strings.ToLower(strings.TrimSpace(input))

	switch {
	case lower == "/help":
		return helpText, false
	case lower == "/persona":
		names := m.status.Personas()
		return fmt.Sprintf("Personas: %s\nTab to cycle", strings.Join(names, ", ")), false
	case lower == "/providers", lower == "/provider":
		return m.providerListMsg(), false
	case strings.HasPrefix(lower, "/provider "):
		name := strings.TrimSpace(input[len("/provider "):])
		return m.switchProviderMsg(name), false
	case strings.HasPrefix(lower, "/model "):
		name := strings.TrimSpace(input[len("/model "):])
		return fmt.Sprintf("Model set to: %s", name), false
	case lower == "/clear":
		return "Chat cleared", true
	case lower == "/quit":
		return "Use Ctrl+Q or Esc to quit", false
	default:
		return fmt.Sprintf("Command not recognized: %s\nType /help for available commands.", input), false
	}
}

func (m *Model) providerListMsg() string {
	if m.router == nil {
		return "No providers configured. Run 'boi setup'."
	}
	stats := m.router.Stats()
	var sb strings.Builder
	sb.WriteString("Providers:\n")
	for _, s := range stats {
		active := " "
		if s.Name == m.router.ActiveProvider() {
			active = "*"
		}
		sb.WriteString(fmt.Sprintf("  %s %s — %d%% available (%d calls, %d ok)\n",
			active, s.Name, s.UsagePct(), s.CallCount, s.SuccessCount))
	}
	return sb.String()
}

func (m *Model) switchProviderMsg(name string) string {
	if m.router == nil {
		return "No providers configured."
	}
	if m.router.SetActiveProvider(name) {
		// Update status bar
		if st := m.router.ActiveStats(); st != nil {
			m.status.SetProvider(name)
			m.status.SetUsagePct(st.AvailablePct())
		}
		return fmt.Sprintf("Switched to provider: %s", name)
	}
	names := m.router.ProviderNames()
	return fmt.Sprintf("Provider '%s' not found. Available: %s", name, strings.Join(names, ", "))
}

func (m *Model) callLLMCmd(input string) tea.Cmd {
	return func() tea.Msg {
		if m.router == nil || m.router.ProviderCount() == 0 {
			return llmResponseMsg{
				content: "No AI providers configured.\nRun 'boi setup' to configure a provider.",
			}
		}

		systemPrompt := m.getSystemPrompt()

		ctx, cancel := context.WithTimeout(context.Background(), 120*time.Second)
		defer cancel()

		resp, err := m.router.Complete(ctx, llm.CompletionRequest{
			Messages: []llm.Message{
				{Role: "system", Content: systemPrompt},
				{Role: "user", Content: input},
			},
			MaxTokens:   4096,
			Temperature: 0.5,
		})
		if err != nil {
			return llmResponseMsg{err: err}
		}
		return llmResponseMsg{
			content:  resp.Content,
			model:    resp.Model,
			provider: resp.Provider,
			tokens:   resp.InputTokens + resp.OutputTokens,
		}
	}
}

func (m *Model) getSystemPrompt() string {
	if m.activeP != nil && m.activeP.SystemPrompt != "" {
		return m.activeP.SystemPrompt
	}
	return "You are a helpful AI assistant. Answer concisely and directly."
}

func (m *Model) handleLLMResponse(msg llmResponseMsg) {
	if msg.err != nil {
		m.chat.AddMessage("error", fmt.Sprintf("LLM error: %v", msg.err))
	} else {
		m.chat.AddAgentMessage(msg.content, msg.provider, msg.model, msg.tokens)
		if msg.provider != "" {
			m.status.SetProvider(msg.provider + "/" + msg.model)
		}
		// Update usage stats from router
		if m.router != nil {
			if st := m.router.ActiveStats(); st != nil {
				m.status.SetUsagePct(st.AvailablePct())
			}
		}
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
  /help      Show this help
  /persona   List available personas
  /clear     Clear chat history
  /quit      Exit BOI CLI
  /provider  Switch LLM provider
  /model     Switch model

Keyboard shortcuts:
  Enter      Send message
  Tab        Switch persona (or autocomplete for /commands)
  Ctrl+N     Newline in input
  Ctrl+L     Clear chat
  Ctrl+Q     Quit
  Esc        Quit
  /          Type / for command palette`
