package tui

import (
	"strings"

	term "github.com/boi-family/boi-cli/internal/platform/terminal"
	"github.com/charmbracelet/bubbles/key"
	"github.com/charmbracelet/bubbles/textarea"
	tea "github.com/charmbracelet/bubbletea"
	"github.com/charmbracelet/lipgloss"
)

const (
	// inputMinHeight is the resting height (inner lines, excl. border).
	inputMinHeight = 3
	// inputMaxHeight is the tallest the box grows before scrolling internally.
	inputMaxHeight = 8
)

// InputModel is a responsive, auto-growing bordered input box.
// The frame (rounded border) always matches the content width exactly, and
// the height expands/collapses with the number of wrapped lines (clamped
// between inputMinHeight and inputMaxHeight) so the shape never breaks.
type InputModel struct {
	textarea textarea.Model
	width    int // full width including border
	height   int // current inner height (excl. border)
}

func NewInput() InputModel {
	ta := textarea.New()
	ta.Placeholder = "Ask anything... (Enter to send, Ctrl+N for newline)"
	ta.CharLimit = 5000
	ta.ShowLineNumbers = false
	ta.MaxHeight = inputMaxHeight
	ta.SetHeight(inputMinHeight)
	ta.SetWidth(40)
	ta.Focus()

	ta.KeyMap.InsertNewline = key.NewBinding(key.WithKeys("ctrl+n"))

	// Match the prompt color with the focused border (teal).
	focused, blurred := textarea.DefaultStyles()
	focused.Prompt = focused.Prompt.Foreground(boiTeal)
	focused.Placeholder = focused.Placeholder.Foreground(lipgloss.Color("240"))
	ta.FocusedStyle = focused
	ta.BlurredStyle = blurred

	return InputModel{textarea: ta, width: 0, height: inputMinHeight}
}

func (i *InputModel) Value() string {
	return i.textarea.Value()
}

func (i *InputModel) SetValue(v string) {
	i.textarea.SetValue(v)
	i.refreshHeight()
	i.textarea.Focus()
}

func (i *InputModel) Reset() {
	i.textarea.Reset()
	i.refreshHeight()
	i.textarea.Focus()
}

// SetWidth sets the total width of the box (including the 2-column border).
// The textarea gets width-2 so the frame aligns flush with the layout.
func (i *InputModel) SetWidth(w int) {
	i.width = w
	i.textarea.SetWidth(max(w-2, 1))
	i.refreshHeight()
}

// Height returns the total rendered height (inner lines + top/bottom border).
func (i *InputModel) Height() int {
	return i.height + 2
}

func (i *InputModel) Focus() {
	i.textarea.Focus()
}

func (i *InputModel) Blur() {
	i.textarea.Blur()
}

// refreshHeight recomputes the inner height from the wrapped line count so the
// box grows as the user types and collapses when text is removed.
func (i *InputModel) refreshHeight() {
	val := i.textarea.Value()
	innerW := max(i.textarea.Width(), 1)

	lines := 0
	if val != "" {
		for _, l := range strings.Split(val, "\n") {
			w := term.ThaiStringWidth(l)
			if w == 0 {
				lines++
			} else {
				lines += (w + innerW - 1) / innerW // ceil(w / innerW)
			}
		}
	}
	// A trailing newline should not add an empty phantom row.
	if strings.HasSuffix(val, "\n") {
		lines--
	}

	h := clamp(lines, inputMinHeight, inputMaxHeight)
	i.textarea.SetHeight(h)
	i.height = h
}

func (i *InputModel) Update(msg tea.Msg) (InputModel, tea.Cmd) {
	var cmd tea.Cmd
	i.textarea, cmd = i.textarea.Update(msg)
	i.refreshHeight()
	return *i, cmd
}

func (i *InputModel) View() string {
	border := InputBorderStyle
	if i.textarea.Focused() {
		border = border.Copy().BorderForeground(boiTeal)
	}
	return border.Width(i.width).Render(i.textarea.View())
}

func clamp(v, lo, hi int) int {
	if v < lo {
		return lo
	}
	if v > hi {
		return hi
	}
	return v
}
