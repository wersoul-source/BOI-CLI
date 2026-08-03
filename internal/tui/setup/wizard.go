package setup

import (
	"fmt"
	"net/http"
	"os"
	"path/filepath"
	"strings"
	"time"

	"github.com/boi-family/boi-cli/internal/registry"
	"github.com/boi-family/boi-cli/internal/workspace"
	"github.com/charmbracelet/bubbles/textinput"
	tea "github.com/charmbracelet/bubbletea"
	"github.com/charmbracelet/lipgloss"
)

// ConfiguredProvider holds the result of configuring one provider.
type ConfiguredProvider struct {
	Name     string
	APIKey   string
	Endpoint string
	Model    string
}

// WizardResult is returned after the setup wizard completes.
type WizardResult struct {
	Providers []ConfiguredProvider
	Cancelled bool
}

type step int

const (
	stepCount   step = iota // ask how many providers
	stepSelect              // select provider from list
	stepAPIKey              // enter API key
	stepModel               // select model
	stepDone                // summary + test + save
)

type WizardModel struct {
	step       step
	width      int
	height     int
	totalCount int
	current    int // 0-indexed current provider

	cursor    int
	listItems []string // display items for selection
	listMode  string   // "provider" or "model"

	textInput textinput.Model

	registry  *registry.Registry
	providers []ConfiguredProvider

	// Current provider being configured
	curName    string
	curLabel   string
	curBaseURL string
	curModel   string
	curCustomText bool // currently entering custom model name via text input

	done      bool
	cancelled bool
	infoMsg   string
	errorMsg  string
}

var (
	listNormalStyle = lipgloss.NewStyle().Foreground(lipgloss.Color("#E2E8F0"))
	listActiveStyle = lipgloss.NewStyle().Foreground(lipgloss.Color("#A78BFA")).Bold(true)
	listCursorStyle = lipgloss.NewStyle().Foreground(lipgloss.Color("#6C63FF")).Bold(true).Render(">")
	titleStyle      = lipgloss.NewStyle().Foreground(lipgloss.Color("#6C63FF")).Bold(true)
	subtitleStyle   = lipgloss.NewStyle().Foreground(lipgloss.Color("#A78BFA"))
	infoMsgStyle    = lipgloss.NewStyle().Foreground(lipgloss.Color("#6B7280"))
	errorMsgStyle   = lipgloss.NewStyle().Foreground(lipgloss.Color("#EF4444")).Bold(true)
	successMsgStyle = lipgloss.NewStyle().Foreground(lipgloss.Color("#22C55E")).Bold(true)
)

func NewWizard(r *registry.Registry) *WizardModel {
	ti := textinput.New()
	ti.Placeholder = "sk-..."
	ti.EchoMode = textinput.EchoPassword
	ti.EchoCharacter = '•'
	ti.Focus()
	ti.CharLimit = 200
	ti.Width = 60

	return &WizardModel{
		step:      stepCount,
		registry:  r,
		textInput: ti,
	}
}

func (m *WizardModel) Init() tea.Cmd {
	return textinput.Blink
}

func (m *WizardModel) Update(msg tea.Msg) (tea.Model, tea.Cmd) {
	switch msg := msg.(type) {
	case tea.WindowSizeMsg:
		m.width = msg.Width
		m.height = msg.Height
		m.textInput.Width = min(60, m.width-10)
		return m, nil

	case tea.KeyMsg:
		if m.step == stepDone {
			m.done = true
			m.cancelled = false
			return m, tea.Quit
		}

		switch msg.String() {
		case "ctrl+c":
			m.cancelled = true
			m.done = true
			return m, tea.Quit
		case "esc":
			return m.goBack()
		}

		switch m.step {
		case stepCount:
			return m.updateCount(msg)
		case stepSelect:
			return m.updateSelect(msg)
		case stepAPIKey:
			return m.updateAPIKey(msg)
		case stepModel:
			return m.updateModel(msg)
		}
	}

	if m.step == stepAPIKey || (m.step == stepModel && m.curCustomText) {
		var cmd tea.Cmd
		m.textInput, cmd = m.textInput.Update(msg)
		return m, cmd
	}

	return m, nil
}

func (m *WizardModel) updateCount(msg tea.KeyMsg) (tea.Model, tea.Cmd) {
	switch msg.String() {
	case "1", "2", "3", "4", "5", "6", "7", "8", "9":
		m.totalCount = int(msg.Runes[0] - '0')
		return m, nil
	case "0":
		if m.totalCount > 0 {
			m.totalCount = m.totalCount * 10
		}
		return m, nil
	case "backspace":
		m.totalCount = m.totalCount / 10
		return m, nil
	case "enter":
		if m.totalCount >= 1 && m.totalCount <= 10 {
			m.cursor = 0
			m.buildProviderList()
			m.step = stepSelect
			m.errorMsg = ""
		} else {
			m.errorMsg = "Please enter 1-10"
		}
		return m, nil
	}
	return m, nil
}

func (m *WizardModel) buildProviderList() {
	m.listMode = "provider"
	m.listItems = m.registry.Labels()
	m.listItems = append(m.listItems, "Other (custom endpoint)")
}

func (m *WizardModel) buildModelList(name string) {
	m.listMode = "model"
	models := m.registry.ModelsFor(name)
	m.listItems = models
	m.listItems = append(m.listItems, "[Type custom model...]")
}

func (m *WizardModel) updateSelect(msg tea.KeyMsg) (tea.Model, tea.Cmd) {
	switch msg.String() {
	case "up", "k":
		if m.cursor > 0 {
			m.cursor--
		}
	case "down", "j":
		if m.cursor < len(m.listItems)-1 {
			m.cursor++
		}
	case "enter":
		if m.listMode == "provider" {
			return m.selectProvider()
		}
		return m, nil
	}
	return m, nil
}

func (m *WizardModel) selectProvider() (tea.Model, tea.Cmd) {
	if m.cursor == len(m.listItems)-1 {
		// "Other" selected
		m.curName = "other"
		m.curLabel = "Custom"
		m.curBaseURL = ""
	} else {
		names := m.registry.Names()
		if m.cursor < len(names) {
			m.curName = names[m.cursor]
			entry := m.registry.Get(m.curName)
			if entry != nil {
				m.curLabel = entry.Label
				m.curBaseURL = entry.BaseURL
			}
		}
	}

	m.textInput.SetValue("")
	m.textInput.Placeholder = "sk-..."
	m.textInput.Focus()
	m.step = stepAPIKey
	return m, textinput.Blink
}

func (m *WizardModel) updateAPIKey(msg tea.KeyMsg) (tea.Model, tea.Cmd) {
	switch msg.String() {
	case "enter":
		key := strings.TrimSpace(m.textInput.Value())
		if key == "" {
			// Skip this provider
			m.current++
			m.errorMsg = ""
			if m.current >= m.totalCount {
				return m.finish()
			}
			m.cursor = 0
			m.buildProviderList()
			m.listMode = "provider"
			m.step = stepSelect
			return m, nil
		}
		// Store API key in textinput, proceed to model selection
		m.infoMsg = "API key set. Select model..."
		m.errorMsg = ""
		m.cursor = 0
		if m.curName == "other" {
			// Custom provider: enter model name as text
			m.curCustomText = true
			m.textInput.SetValue("")
			m.textInput.Placeholder = "e.g. gpt-4o"
			m.textInput.EchoMode = textinput.EchoNormal
			m.textInput.Focus()
			m.step = stepModel
		} else {
			m.buildModelList(m.curName)
			m.step = stepModel
		}
		return m, nil
	}
	// Forward all other keys to textinput (let user type the API key)
	var cmd tea.Cmd
	m.textInput, cmd = m.textInput.Update(msg)
	return m, cmd
}

func (m *WizardModel) updateModel(msg tea.KeyMsg) (tea.Model, tea.Cmd) {
	if m.curCustomText {
		// Custom model text input mode
		switch msg.String() {
		case "enter":
			customModel := strings.TrimSpace(m.textInput.Value())
			if customModel == "" {
				customModel = "custom"
			}
			return m.confirmProvider(customModel)
		}
		return m, nil
	}

	switch msg.String() {
	case "up", "k":
		if m.cursor > 0 {
			m.cursor--
		}
	case "down", "j":
		if m.cursor < len(m.listItems)-1 {
			m.cursor++
		}
	case "enter":
		return m.selectModel()
	}
	return m, nil
}

func (m *WizardModel) selectModel() (tea.Model, tea.Cmd) {
	if m.cursor == len(m.listItems)-1 {
		// Custom model: switch to text input mode
		m.curCustomText = true
		m.textInput.SetValue("")
		m.textInput.Placeholder = "Enter model name..."
		m.textInput.EchoMode = textinput.EchoNormal
		m.textInput.Focus()
		return m, nil
	}

	return m.confirmProvider(m.listItems[m.cursor])
}

// confirmProvider saves the provider with the given model and advances.
func (m *WizardModel) confirmProvider(model string) (tea.Model, tea.Cmd) {
	key := m.textInput.Value()
	m.providers = append(m.providers, ConfiguredProvider{
		Name:     m.curName,
		APIKey:   key,
		Endpoint: m.curBaseURL,
		Model:    model,
	})

	m.current++
	m.curCustomText = false

	if m.current >= m.totalCount {
		return m.finish()
	}

	// Reset for next provider
	m.cursor = 0
	m.textInput.SetValue("")
	m.textInput.Placeholder = "sk-..."
	m.textInput.EchoMode = textinput.EchoPassword
	m.buildProviderList()
	m.listMode = "provider"
	m.step = stepSelect
	m.infoMsg = fmt.Sprintf("✓ Provider %d configured. Select next provider...", m.current)
	return m, nil
}

func (m *WizardModel) finish() (tea.Model, tea.Cmd) {
	m.testAllProviders()
	m.step = stepDone
	m.done = true
	m.cancelled = false
	return m, nil
}

func (m *WizardModel) testAllProviders() {
	for _, p := range m.providers {
		err := testProviderEndpoint(p)
		if err != nil {
			m.errorMsg = fmt.Sprintf("⚠ %s: %v", p.Name, err)
		}
	}
	m.infoMsg = fmt.Sprintf("✓ %d provider(s) configured. Writing .env...", len(m.providers))
}

func testProviderEndpoint(p ConfiguredProvider) error {
	if p.Endpoint == "" {
		return nil // skip test for custom
	}
	url := strings.TrimRight(p.Endpoint, "/") + "/models"
	req, err := http.NewRequest("GET", url, nil)
	if err != nil {
		return fmt.Errorf("unreachable")
	}
	req.Header.Set("Authorization", "Bearer "+p.APIKey)
	client := &http.Client{Timeout: 8 * time.Second}
	resp, err := client.Do(req)
	if err != nil {
		return fmt.Errorf("unreachable")
	}
	resp.Body.Close()
	if resp.StatusCode == 200 || resp.StatusCode == 401 || resp.StatusCode == 403 {
		if resp.StatusCode != 200 {
			return fmt.Errorf("invalid API key")
		}
		return nil
	}
	return nil
}

func (m *WizardModel) writeEnv() error {
	var lines []string
	lines = append(lines, "# BOI CLI — Provider Supply Chain")
	lines = append(lines, "# Generated by 'boi setup'")
	lines = append(lines, "")
	for i, p := range m.providers {
		n := i + 1
		lines = append(lines, fmt.Sprintf("# Provider %d: %s", n, p.Name))
		lines = append(lines, fmt.Sprintf("PSC_%d_NAME=%s", n, p.Name))
		lines = append(lines, fmt.Sprintf("PSC_%d_API_KEY=%s", n, p.APIKey))
		lines = append(lines, fmt.Sprintf("PSC_%d_BASE_URL=%s", n, p.Endpoint))
		lines = append(lines, fmt.Sprintf("PSC_%d_MODEL=%s", n, p.Model))
		lines = append(lines, "")
	}
	content := strings.Join(lines, "\n")

	// Write to project root (workspace), not cwd
	root, err := workspace.DetectRoot()
	var envPath string
	if err == nil {
		envPath = filepath.Join(root, ".env")
	} else {
		// Fallback to cwd if no workspace found
		cwd, _ := os.Getwd()
		envPath = filepath.Join(cwd, ".env")
	}
	return os.WriteFile(envPath, []byte(content), 0644)
}

func (m *WizardModel) View() string {
	if m.width == 0 {
		return "Loading..."
	}

	border := lipgloss.NewStyle().
		Border(lipgloss.RoundedBorder()).
		BorderForeground(lipgloss.Color("#6C63FF")).
		Width(min(70, m.width-4)).
		Padding(1, 2)

	var body string

	switch m.step {
	case stepCount:
		body = m.viewCount()
	case stepSelect:
		body = m.viewSelect()
	case stepAPIKey:
		body = m.viewAPIKey()
	case stepModel:
		body = m.viewModel()
	case stepDone:
		body = m.viewDone()
	}

	if m.infoMsg != "" {
		body += "\n\n" + successMsgStyle.Render("  "+m.infoMsg)
	}
	if m.errorMsg != "" {
		body += "\n\n" + errorMsgStyle.Render("  "+m.errorMsg)
	}

	body += m.viewHelp()

	bordered := border.Render(body)
	return lipgloss.Place(m.width, m.height, lipgloss.Center, lipgloss.Center, bordered)
}

func (m *WizardModel) viewCount() string {
	title := titleStyle.Render("⚡ BOI CLI — Provider Setup")
	subtitle := subtitleStyle.Render("Configure your AI providers for auto-fallback")
	question := "\n\n  How many providers?  " + listCursorStyle + " " +
		lipgloss.NewStyle().Foreground(lipgloss.Color("#22C55E")).Bold(true).Render(fmt.Sprintf("[%d]", m.totalCount))
	hint := infoMsgStyle.Render("  Type a number 1-10, then press Enter")

	return title + "\n  " + subtitle + question + "\n\n" + hint
}

func (m *WizardModel) viewSelect() string {
	header := fmt.Sprintf("  Provider %d of %d", m.current+1, m.totalCount)
	title := titleStyle.Render("⚡ Select Provider") + infoMsgStyle.Render("  "+header)

	var items strings.Builder
	for i, item := range m.listItems {
		if i == m.cursor {
			items.WriteString(fmt.Sprintf("\n  %s %s", listCursorStyle, listActiveStyle.Render(item)))
		} else {
			items.WriteString(fmt.Sprintf("\n   %s", listNormalStyle.Render(item)))
		}
	}

	return title + items.String()
}

func (m *WizardModel) viewAPIKey() string {
	header := fmt.Sprintf("  Provider %d of %d: %s", m.current+1, m.totalCount, m.curLabel)
	title := titleStyle.Render("⚡ Enter API Key") + infoMsgStyle.Render("  "+header)

	endpoint := fmt.Sprintf("\n\n  Endpoint: %s", m.curBaseURL)

	apiKeySection := fmt.Sprintf("\n\n  API Key:\n  %s", m.textInput.View())

	return title + endpoint + apiKeySection
}

func (m *WizardModel) viewModel() string {
	header := fmt.Sprintf("  Provider %d of %d: %s", m.current+1, m.totalCount, m.curLabel)
	title := titleStyle.Render("⚡ Select Model") + infoMsgStyle.Render("  "+header)

	if m.curCustomText {
		customPrompt := fmt.Sprintf("\n\n  Enter model name:\n  %s", m.textInput.View())
		return title + customPrompt
	}

	var items strings.Builder
	for i, item := range m.listItems {
		if i == m.cursor {
			items.WriteString(fmt.Sprintf("\n  %s %s", listCursorStyle, listActiveStyle.Render(item)))
		} else {
			items.WriteString(fmt.Sprintf("\n   %s", listNormalStyle.Render(item)))
		}
	}

	return title + items.String()
}

func (m *WizardModel) viewDone() string {
	title := titleStyle.Render("⚡ Setup Complete")

	var sb strings.Builder
	sb.WriteString(title)
	sb.WriteString("\n")
	for i, p := range m.providers {
		sb.WriteString(fmt.Sprintf("\n  %d. %s — %s", i+1, successMsgStyle.Render(p.Name), subtitleStyle.Render(p.Model)))
	}

	sb.WriteString("\n\n  " + successMsgStyle.Render("✓ .env file created"))
	sb.WriteString("\n  " + infoMsgStyle.Render("Run 'boi' to start!"))

	return sb.String()
}

func (m *WizardModel) viewHelp() string {
	switch m.step {
	case stepCount:
		return "\n\n  " + infoMsgStyle.Render("Enter: confirm    Esc: cancel    Ctrl+C: quit")
	case stepSelect, stepModel:
		return "\n\n  " + infoMsgStyle.Render("↑↓: navigate    Enter: select    Esc: back    Ctrl+C: quit")
	case stepAPIKey:
		return "\n\n  " + infoMsgStyle.Render("Enter: confirm    Esc: back    Ctrl+C: quit")
	case stepDone:
		return "\n\n  " + infoMsgStyle.Render("Press any key to exit")
	}
	return ""
}

// goBack moves to the previous step, preserving user input.
func (m *WizardModel) goBack() (tea.Model, tea.Cmd) {
	switch m.step {
	case stepSelect:
		// Back to count
		m.step = stepCount
		m.errorMsg = ""
		m.infoMsg = ""
	case stepAPIKey:
		// Back to provider selection (keep provider name)
		m.cursor = 0
		m.buildProviderList()
		m.listMode = "provider"
		m.step = stepSelect
		m.errorMsg = ""
		m.infoMsg = ""
	case stepModel:
		// Back to API key (keep the key in textInput)
		if m.curCustomText {
			m.curCustomText = false
		}
		m.textInput.SetValue(m.textInput.Value()) // keep current value
		m.textInput.EchoMode = textinput.EchoPassword
		m.textInput.Placeholder = "sk-..."
		m.step = stepAPIKey
		m.errorMsg = ""
		m.infoMsg = ""
		m.textInput.Focus()
	case stepCount:
		// At the beginning — cancel
		m.cancelled = true
		m.done = true
		return m, tea.Quit
	}
	return m, textinput.Blink
}

// Result returns the configured providers (nil if not done).
func (m *WizardModel) Result() *WizardResult {
	if !m.done {
		return nil
	}
	return &WizardResult{
		Providers: m.providers,
		Cancelled: m.cancelled,
	}
}

// Run runs the setup wizard TUI and returns the result.
func Run(r *registry.Registry) *WizardResult {
	w := NewWizard(r)
	// Note: WithoutAltScreen for better compatibility with PowerShell 7 / conpty
	p := tea.NewProgram(w)
	model, err := p.Run()
	if err != nil {
		return &WizardResult{Cancelled: true}
	}
	if wm, ok := model.(*WizardModel); ok {
		if res := wm.Result(); res != nil {
			if !res.Cancelled && len(res.Providers) > 0 {
				wm.writeEnv()
			}
			return res
		}
	}
	return &WizardResult{Cancelled: true}
}
