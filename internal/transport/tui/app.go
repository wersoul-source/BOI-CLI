package tui

import (
	"context"
	"errors"
	"fmt"
	"os"
	"path/filepath"
	"strings"
	"time"

	"github.com/boi-family/boi-cli/internal/app"
	coreblock "github.com/boi-family/boi-cli/internal/core"
	"github.com/boi-family/boi-cli/internal/core/persona"
	"github.com/boi-family/boi-cli/internal/equipment/capability"
	"github.com/boi-family/boi-cli/internal/equipment/tools/filesystem"
	"github.com/boi-family/boi-cli/internal/runtime/agent"
	llm "github.com/boi-family/boi-cli/internal/runtime/llm"
	"github.com/boi-family/boi-cli/internal/runtime/workspace"
	"github.com/boi-family/boi-cli/internal/service/config"
	llmfactory "github.com/boi-family/boi-cli/internal/service/provider/factory"
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
	taskID   string
	manifest string
}

type approvalRequestedMsg struct {
	request   agent.ApprovalRequest
	decisions chan<- agent.ApprovalDecision
}

type approvalDecisionSentMsg struct {
	decision agent.ApprovalDecision
	err      error
}

type runtimeProgressMsg struct{ event agent.EngineEvent }

// streamTextMsg carries the reply produced so far. The text is cumulative, so
// the newest message is the whole answer and earlier ones can be ignored.
type streamTextMsg struct{ text string }

type Model struct {
	// composeErr explains why no Agent could be composed, e.g. a missing
	// required Block; it is shown when the chat opens.
	composeErr      error
	splash          *SplashModel
	chat            ChatModel
	input           InputModel
	status          StatusModel
	help            HelpModel
	approval        ApprovalModel
	width           int
	height          int
	mode            string
	router          *llm.Router
	activeP         *persona.Persona
	root            string
	sandbox         *workspace.Sandbox
	workspaceReader *filesystem.Reader
	agentService    *agent.Service
	cancelActive    context.CancelFunc
	approvalSink    chan<- agent.ApprovalDecision
	runtimeEvents   <-chan agent.RuntimeEvent
	boiDir          string
	environment     coreblock.AgentEnvironment
	agent           *app.Agent
	configured      []llmfactory.ConfiguredProvider
}

// cliName is the command the user typed to start this program, so hints name
// a command that exists on their machine (boi-agent, boi, boi-agent.exe).
var cliName = func() string {
	name := strings.TrimSuffix(filepath.Base(os.Args[0]), ".exe")
	if name == "" || name == "." || strings.HasSuffix(name, ".test") {
		return "boi-agent"
	}
	return name
}()

func NewApp(runtime *app.Runtime) *Model {
	agentName := coreblock.DefaultAgentName
	provider := "none"
	root := runtime.WorkspaceRoot

	if root != "" {
		if identity, identityErr := coreblock.LoadIdentity(runtime.IdentityPath); identityErr == nil {
			agentName = identity.Name
		}
		cfg, cfgErr := config.LoadFrom(runtime.ConfigPath)
		if cfgErr == nil && cfg.Provider != "" {
			if cfg.Model != "" {
				provider = fmt.Sprintf("%s/%s", cfg.Provider, cfg.Model)
			} else {
				provider = cfg.Provider
			}
		}
	}

	boiDir := filepath.Join(root, ".boi")
	memoryCount := countMemoryEntries(filepath.Join(boiDir, "memory"))
	skillList, skillCount := indexedSkillNames(boiDir)
	providerCount := 0

	// Compose the Agent once through the application composition root.
	configuredProviders, _ := llmfactory.LoadConfiguredProvidersFromEnv()
	composed, composeErr := runtime.BuildAgent(configuredProviders)
	var router *llm.Router
	var agentService *agent.Service
	var environment coreblock.AgentEnvironment
	if composeErr == nil {
		router, agentService, environment = composed.Router, composed.Service, composed.Environment
		providerCount = len(composed.Qualified)
	}
	if providerCount == 0 && len(configuredProviders) > 0 {
		provider = "unqualified"
	} else if providerCount > 0 {
		provider = composed.Qualified[0].Name + "/" + composed.Qualified[0].Model
	}
	splash := NewSplash(root, agentName, coreblock.CorePersonaName, providerCount, memoryCount, skillCount, strings.Join(skillList, ", "), runtime.Version)

	// Runtime persona is a Core invariant. The user names the Agent, not the persona.
	activeP := persona.CorePersona()

	m := &Model{
		splash:          splash,
		chat:            NewChat(),
		input:           NewInput(),
		status:          NewStatus(agentName, provider, nil),
		help:            NewHelp(),
		approval:        NewApproval(),
		mode:            "splash",
		router:          router,
		activeP:         activeP,
		root:            root,
		sandbox:         runtime.Sandbox,
		workspaceReader: filesystem.NewReader(runtime.Sandbox),
		agentService:    agentService,
		agent:           composed,
		boiDir:          runtime.BoiDir,
		environment:     environment,
		composeErr:      composeErr,
		configured:      configuredProviders,
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
	m.approval.SetWidth(m.width)
	m.chat.SetSize(m.width, m.chatHeight())
	m.input.SetWidth(m.width)
	m.chat.AddMessage("system", "BOI CLI ready. Type /help for commands.")
	if m.composeErr != nil {
		m.chat.AddMessage("system", "Agent unavailable: "+m.composeErr.Error())
	}
	return tickCmd()
}

// chatHeight computes the chat viewport height from the real layout:
// total - status bar - input box (grows/shrinks) - help bar - 1 breathing row.
func (m *Model) chatHeight() int {
	controlHeight := m.input.Height()
	if m.approval.Active() {
		controlHeight = m.approval.Height()
	}
	h := m.height - m.status.Height() - controlHeight - m.help.Height() - 1
	if h < 1 {
		h = 1
	}
	return h
}

func (m *Model) isBusy() bool {
	status := m.status.Status()
	return status == "thinking" || status == "working" || status == "cancelling" || status == "approval"
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
		m.approval.SetWidth(msg.Width)
		m.chat.SetSize(msg.Width, m.chatHeight())
		m.input.SetWidth(msg.Width)

	case tickMsg:
		m.status.Tick()
		if m.approval.Active() {
			if decision, expired := m.approval.Decide("", time.Time(msg)); expired {
				cmds = append(cmds, m.finishApproval(*decision))
			}
		}
		cmds = append(cmds, tickCmd())

	case agentResponseMsg:
		m.handleAgentResponse(msg)
		return m, tea.Batch(cmds...)

	case workspaceResponseMsg:
		m.handleWorkspaceResponse(msg)
		return m, tea.Batch(cmds...)

	case approvalRequestedMsg:
		if err := m.openApproval(msg); err != nil {
			m.chat.AddMessage("error", fmt.Sprintf("Approval request rejected: %v", err))
			m.status.SetStatus("error")
		}
		return m, nil

	case approvalDecisionSentMsg:
		if msg.err != nil {
			m.chat.AddMessage("error", fmt.Sprintf("Approval decision was not delivered: %v", msg.err))
			m.status.SetStatus("error")
			return m, nil
		}
		m.chat.AddMessage("system", approvalDecisionSummary(msg.decision))
		m.status.SetStatus("thinking")
		return m, waitAgentEventCmd(m.runtimeEvents)

	case streamTextMsg:
		m.chat.SetPartial(msg.text)
		m.status.SetStatus("writing")
		return m, waitAgentEventCmd(m.runtimeEvents)

	case runtimeProgressMsg:
		switch msg.event.Phase {
		case agent.PhaseAct:
			m.status.SetStatus("working")
		case agent.PhaseStopped:
		default:
			m.status.SetStatus("thinking")
		}
		return m, waitAgentEventCmd(m.runtimeEvents)

	case tea.KeyMsg:
		if msg.String() == "ctrl+q" {
			if m.cancelActive != nil {
				m.cancelActive()
			}
			return m, tea.Quit
		}
		if m.approval.Active() {
			decision, handled := m.approval.Decide(msg.String(), time.Now())
			if !handled {
				return m, nil
			}
			return m, m.finishApproval(*decision)
		}
		switch msg.String() {
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
			// Tab is reserved for command completion. Core Persona cannot be switched.
			if strings.HasPrefix(m.input.Value(), "/") {
				best := m.help.BestMatch(m.input.Value())
				if best != "" && best != m.input.Value() {
					m.input.SetValue(best)
					m.help.SetSuggestions(best)
				}
			}
			return m, nil

		case "ctrl+l":
			m.chat.Clear()
			if m.agentService != nil {
				m.agentService.ResetConversation()
			}
			m.chat.AddMessage("system", "Chat and conversation memory cleared")
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
		return fmt.Sprintf("Core Persona: %s (fixed)\nAgent: %s", coreblock.CorePersonaName, m.status.AgentName()), false
	case lower == "/providers", lower == "/provider":
		return m.providerListMsg(), false
	case strings.HasPrefix(lower, "/provider "):
		name := strings.TrimSpace(input[len("/provider "):])
		return m.switchProviderMsg(name), false
	case lower == "/model", strings.HasPrefix(lower, "/model "):
		return m.modelMsg(), false
	case lower == "/config", lower == "/setup", lower == "/quickstart":
		return m.configMsg(), false
	case lower == "/clear":
		if m.agentService != nil {
			m.agentService.ResetConversation()
		}
		return "Chat and conversation memory cleared", true
	case lower == "/quit", lower == "/exit":
		return "Use Ctrl+Q or Esc to quit", false
	default:
		return fmt.Sprintf("Command not recognized: %s\nType /help for available commands.", input), false
	}
}

// noProviderMsg explains why the Agent cannot answer and the one step that
// fixes it. A configured but unqualified provider is a different problem from
// no provider at all.
func (m *Model) noProviderMsg() string {
	if len(m.configured) > 0 {
		names := make([]string, 0, len(m.configured))
		for _, p := range m.configured {
			names = append(names, p.Name+"/"+p.Model)
		}
		return fmt.Sprintf("A model is configured (%s) but has not been checked in this folder.\nRun '%s quickstart' in a terminal, in the folder you work in.", strings.Join(names, ", "), cliName)
	}
	return fmt.Sprintf("No model is configured for %s.\nRun '%s quickstart' in a terminal, in the folder you work in; it saves the choice in .env here.", m.root, cliName)
}

func (m *Model) modelMsg() string {
	if m.agent == nil || len(m.agent.Qualified) == 0 {
		return m.noProviderMsg()
	}
	active := m.agent.Qualified[0]
	if m.router != nil {
		for _, q := range m.agent.Qualified {
			if q.Name == m.router.ActiveProvider() {
				active = q
			}
		}
	}
	return fmt.Sprintf("Model: %s\nProvider: %s\nAddress: %s\nThe model is chosen in '%s quickstart --change'; it cannot be switched from inside the chat.", active.Model, active.Name, active.BaseURL, cliName)
}

func (m *Model) configMsg() string {
	var sb strings.Builder
	fmt.Fprintf(&sb, "Folder: %s\n", m.root)
	if m.agent == nil || len(m.agent.Qualified) == 0 {
		sb.WriteString(m.noProviderMsg())
		return sb.String()
	}
	for _, q := range m.agent.Qualified {
		fmt.Fprintf(&sb, "Model: %s/%s at %s\n", q.Name, q.Model, q.BaseURL)
	}
	fmt.Fprintf(&sb, "To change the model or the key: '%s quickstart --change' in a terminal.", cliName)
	return sb.String()
}

func (m *Model) providerListMsg() string {
	if m.router == nil || len(m.router.Stats()) == 0 {
		return m.noProviderMsg()
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
		return m.noProviderMsg()
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
	if m.agentService == nil {
		return func() tea.Msg { return agentResponseMsg{err: fmt.Errorf("agent service is not configured")} }
	}
	if _, err := m.agent.PrepareTask(input); err != nil {
		return func() tea.Msg { return agentResponseMsg{err: fmt.Errorf("select capability registry: %w", err)} }
	}
	m.runtimeEvents = m.agentService.Start(ctx, input)
	return waitAgentEventCmd(m.runtimeEvents)
}

func indexedSkillNames(boiDir string) ([]string, int) {
	index, err := capability.LoadIndex(capability.IndexPath(boiDir, capability.KindSkill), capability.KindSkill)
	if err != nil {
		return nil, 0
	}
	var names []string
	for _, entry := range index.Entries {
		if entry.Enabled {
			names = append(names, entry.Name)
		}
	}
	return names, len(names)
}

func waitAgentEventCmd(events <-chan agent.RuntimeEvent) tea.Cmd {
	return func() tea.Msg {
		if events == nil {
			return agentResponseMsg{err: fmt.Errorf("agent event stream is not configured")}
		}
		event, ok := <-events
		if !ok {
			return agentResponseMsg{err: fmt.Errorf("agent event stream closed without a result")}
		}
		if event.Approval != nil {
			return approvalRequestedMsg{request: event.Approval.Request, decisions: event.Approval.Decisions}
		}
		if event.Partial != "" {
			return streamTextMsg{text: event.Partial}
		}
		if event.Engine != nil {
			return runtimeProgressMsg{event: *event.Engine}
		}
		if event.Err != nil {
			return agentResponseMsg{err: event.Err}
		}
		if event.Result == nil {
			return agentResponseMsg{err: fmt.Errorf("agent event is empty")}
		}
		content := event.Result.Response
		if event.Result.Notice != "" {
			content += "\n\n" + event.Result.Notice
		}
		return agentResponseMsg{content: content, model: event.Result.Model, provider: event.Result.Provider, tokens: event.Result.Tokens, taskID: event.Result.TaskID, manifest: event.Result.Manifest}
	}
}

func (m *Model) handleAgentResponse(msg agentResponseMsg) {
	if m.cancelActive != nil {
		m.cancelActive()
		m.cancelActive = nil
	}
	m.runtimeEvents = nil
	// The turn is over, so the live reply is replaced by the real message
	// below, or dropped when the turn failed.
	m.chat.ClearPartial()
	if errors.Is(msg.err, context.Canceled) {
		m.chat.AddMessage("system", "Agent task cancelled.")
		m.status.SetStatus("idle")
		return
	}
	if msg.err != nil {
		if errors.Is(msg.err, agent.ErrNoProvider) {
			m.chat.AddMessage("error", m.noProviderMsg())
		} else {
			m.chat.AddMessage("error", fmt.Sprintf("Agent error: %v", msg.err))
		}
	} else {
		m.chat.AddAgentMessage(msg.content, msg.provider, msg.model, msg.tokens)
		if msg.manifest != "" {
			m.chat.AddMessage("system", fmt.Sprintf("Task %s · Manifest: %s", msg.taskID, msg.manifest))
		}
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

func (m *Model) openApproval(msg approvalRequestedMsg) error {
	if m.approval.Active() {
		return fmt.Errorf("another approval request is already active")
	}
	if msg.decisions == nil {
		return fmt.Errorf("approval decision channel is not configured")
	}
	if err := m.approval.Open(msg.request, time.Now()); err != nil {
		return err
	}
	m.approvalSink = msg.decisions
	m.input.Blur()
	m.status.SetStatus("approval")
	m.chat.SetSize(m.width, m.chatHeight())
	return nil
}

func (m *Model) finishApproval(decision agent.ApprovalDecision) tea.Cmd {
	sink := m.approvalSink
	m.approval.Close()
	m.approvalSink = nil
	m.input.Focus()
	m.chat.SetSize(m.width, m.chatHeight())
	return sendApprovalDecisionCmd(sink, decision)
}

func sendApprovalDecisionCmd(
	sink chan<- agent.ApprovalDecision,
	decision agent.ApprovalDecision,
) tea.Cmd {
	return func() tea.Msg {
		if err := decision.Validate(); err != nil {
			return approvalDecisionSentMsg{decision: decision, err: err}
		}
		if sink == nil {
			return approvalDecisionSentMsg{decision: decision, err: fmt.Errorf("approval decision channel is not configured")}
		}
		select {
		case sink <- decision:
			return approvalDecisionSentMsg{decision: decision}
		default:
			return approvalDecisionSentMsg{decision: decision, err: fmt.Errorf("approval decision receiver is not ready")}
		}
	}
}

func (m *Model) View() string {
	if m.mode == "splash" {
		return m.splash.View()
	}

	status := m.status.View()
	chat := m.chat.View()
	control := m.input.View()
	if m.approval.Active() {
		control = m.approval.View()
	}
	help := m.help.View()

	return lipgloss.JoinVertical(lipgloss.Top,
		status,
		chat,
		control,
		help,
	)
}

var helpText = `Available commands:
  /help      Show this help
  /persona   Show the fixed Core Persona and Agent name
  /clear     Clear chat history
  /quit      Exit BOI CLI
  /provider  List providers, or /provider NAME to switch
  /model     Show the model in use
  /config    Show the folder, model and how to change it
  /workspace Show the folders the Agent may use
  /allow PATH Allow a folder for the next task only
  /ls [path] List a directory inside the workspace
  /read PATH Read a text file inside the workspace

Keyboard shortcuts:
  Enter      Send message
  Tab        Autocomplete /commands
  Ctrl+N     Newline in input
  Ctrl+L     Clear chat
  Ctrl+Q     Quit
  Esc        Cancel the active Agent task, or quit when idle
  /          Type / for command palette`
