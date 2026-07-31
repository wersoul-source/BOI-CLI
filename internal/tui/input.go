package tui

import (
	"github.com/charmbracelet/bubbles/key"
	"github.com/charmbracelet/bubbles/textarea"
	tea "github.com/charmbracelet/bubbletea"
)

type InputModel struct {
	textarea textarea.Model
}

func NewInput() InputModel {
	ta := textarea.New()
	ta.Placeholder = "Ask anything... (Enter to send, Ctrl+N for newline)"
	ta.CharLimit = 5000
	ta.SetHeight(3)
	ta.Focus()
	ta.ShowLineNumbers = false

	ta.KeyMap.InsertNewline = key.NewBinding(key.WithKeys("ctrl+n"))

	return InputModel{textarea: ta}
}

func (i *InputModel) Value() string {
	return i.textarea.Value()
}

func (i *InputModel) Reset() {
	i.textarea.Reset()
	i.textarea.Focus()
}

func (i *InputModel) SetWidth(w int) {
	i.textarea.SetWidth(w - 4)
}

func (i *InputModel) Focus() {
	i.textarea.Focus()
}

func (i *InputModel) Blur() {
	i.textarea.Blur()
}

func (i *InputModel) Update(msg tea.Msg) (InputModel, tea.Cmd) {
	var cmd tea.Cmd
	i.textarea, cmd = i.textarea.Update(msg)
	return *i, cmd
}

func (i *InputModel) View() string {
	return i.textarea.View()
}
