package tui

import (
	"context"
	"errors"
	"fmt"
	"path/filepath"
	"strings"
	"time"

	"github.com/boi-family/boi-cli/internal/agent"
	"github.com/boi-family/boi-cli/internal/app"
	"github.com/boi-family/boi-cli/internal/config"
	"github.com/boi-family/boi-cli/internal/memory"
	"github.com/boi-family/boi-cli/internal/persona"
	llm "github.com/boi-family/boi-cli/internal/provider"
	llmfactory "github.com/boi-family/boi-cli/internal/provider/factory"
	"github.com/boi-family/boi-cli/internal/tool/filesystem"
	"github.com/boi-family/boi-cli/internal/workspace"
	tea "github.com/charmbracelet/bubbletea"
	"github.com/charmbracelet/lipgloss"
)

type tickMsg time.Time

type agentResponseMsg struct {
	content  string
	err      error
	provider string
	model    string
	tokens   int
}

type Model struct {
	splash          *SplashModel
	chat            ChatModel
	input           InputModel
	status          StatusModel
	help            HelpModel
	width           int
	height          int
	mode            string
	router          *llm.Router
	activeP         *persona.Persona
	root            string
	workspaceReader *filesystem.Reader
	agentService    *agent.Service
	cancelActive    context.CancelFunc
}

func NewApp(runtime *app.Runtime) *Model {
	var personaNames []string
	activePersona := "kamkaew"
	provider := "none"
	root := runtime.WorkspaceRoot

	if root != "" {
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

	splash := NewSplash(root, len(personaNames), strings.Join(personaNames, ", "), providerCount, memoryCount, skillCount, strings.Join(skillList, ", "), runtime.Version)

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
	var memoryHook *memory.MemoryHook
	if store, storeErr := memory.Open(filepath.Join(runtime.BoiDir, "memory")); storeErr == nil {
		memoryHook = memory.NewMemoryHook(store, &memory.SimpleExtractor{})
	}
	agentService := agent.NewService(activeP, router, memoryHook, runtime.Sandbox)

	m := &Model{
		splash:          splash,
		chat:            NewChat(),
		input:           NewInput(),
		status:          NewStatus(activePersona, provider, personaNames),
		help:            NewHelp(),
		mode:            "splash",
		router:          router,
		activeP:         activeP,
		root:            root,
		workspaceReader: filesystem.NewReader(runtime.Sandbox),
		agentService:    agentService,
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

func (m *Model) isBusy() bool {
	status := m.status.Status()
	return status == "thinking" || status == "working" || status == "cancelling"
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

	case agentResponseMsg:
		m.handleAgentResponse(msg)
		return m, tea.Batch(cmds...)

	case workspaceResponseMsg:
		m.handleWorkspaceResponse(msg)
		return m, tea.Batch(cmds...)

	case tea.KeyMsg:
		switch msg.String() {
		case "ctrl+q":
			if m.cancelActive != nil {
				m.cancelActive()
			}
			return m, tea.Quit
		case "ctrl+c", "esc":
			if m.cancelActive != nil {
				m.cancelActive()
				m.status.SetStatus("cancelling")
				m.chat.AddMessage("system", "Cancelling active Agent task...")
				return m, nil
			}
			if m.isBusy() {
				m.chat.AddMessage("system", "The bounded workspace operation is finishing; it cannot be cancelled.")
				return m, nil
			}
			return m, tea.Quit

		case "enter":
			if m.isBusy() {
				m.chat.AddMessage("system", "A task is already running. Press Esc to cancel it.")
				return m, nil
			}
			val := strings.TrimSpace(m.input.Value())
			if val != "" {
				isCmd := strings.HasPrefix(val, "/")
				if !isCmd {
					m.chat.AddMessage("user", val)
				}
				m.input.Reset()
				m.help.SetSuggestions("")
				if isCmd {
					if isWorkspaceCommand(val) {
						m.status.SetStatus("working")
						return m, m.callWorkspaceCmd(val)
					}
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
				cmds = append(cmds, m.startAgentCmd(val))
			}
			return m, tea.Batch(cmds...)

		case "tab":
			if m.isBusy() {
				return m, nil
			}
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
							m.agentService.SetPersona(p)
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

func (m *Model) startAgentCmd(input string) tea.Cmd {
	ctx, cancel := context.WithTimeout(context.Background(), 120*time.Second)
	m.cancelActive = cancel
	return func() tea.Msg {
		if m.agentService == nil {
			return agentResponseMsg{err: fmt.Errorf("agent service is not configured")}
		}
		result, err := m.agentService.Run(ctx, input)
		if err != nil {
			return agentResponseMsg{err: err}
		}
		return agentResponseMsg{
			content:  result.Response,
			model:    result.Model,
			provider: result.Provider,
			tokens:   result.Tokens,
		}
	}
}

func (m *Model) handleAgentResponse(msg agentResponseMsg) {
	if m.cancelActive != nil {
		m.cancelActive()
		m.cancelActive = nil
	}
	if errors.Is(msg.err, context.Canceled) {
		m.chat.AddMessage("system", "Agent task cancelled.")
		m.status.SetStatus("idle")
		return
	}
	if msg.err != nil {
		if errors.Is(msg.err, agent.ErrNoProvider) {
			m.chat.AddMessage("error", "No AI providers configured. Run 'boi setup' to configure a provider.")
		} else {
			m.chat.AddMessage("error", fmt.Sprintf("Agent error: %v", msg.err))
		}
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
  /workspace Show the active sandbox root
  /ls [path] List a directory inside the workspace
  /read PATH Read a text file inside the workspace

Keyboard shortcuts:
  Enter      Send message
  Tab        Switch persona (or autocomplete for /commands)
  Ctrl+N     Newline in input
  Ctrl+L     Clear chat
  Ctrl+Q     Quit
  Esc        Cancel the active Agent task, or quit when idle
  /          Type / for command palette`
