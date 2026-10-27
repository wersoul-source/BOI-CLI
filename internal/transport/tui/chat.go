package tui

import (
	"fmt"
	"strings"
	"time"

	term "github.com/boi-family/boi-cli/internal/platform/terminal"
	"github.com/charmbracelet/bubbles/viewport"
	tea "github.com/charmbracelet/bubbletea"
)

type Message struct {
	Role     string
	Content  string
	Provider string // for agent messages
	Model    string
	Tokens   int
	Time     time.Time
}

type ChatModel struct {
	viewport viewport.Model
	messages []Message
	// partial is the reply being written right now. It is drawn as a bubble
	// after the finished messages and replaced by the real message when the
	// turn ends, so the user reads the answer as it arrives instead of after.
	partial string
	width   int
	height  int
}

func NewChat() ChatModel {
	vp := viewport.New(80, 20)
	return ChatModel{viewport: vp}
}

func (c *ChatModel) AddMessage(role, content string) {
	msg := Message{Role: role, Content: content, Time: time.Now()}
	c.messages = append(c.messages, msg)
	c.renderMessages()
	c.viewport.GotoBottom()
}

// SetPartial shows the reply being written. The text is cumulative, so each
// update replaces the last.
func (c *ChatModel) SetPartial(text string) {
	if c.partial == text {
		return
	}
	c.partial = text
	c.renderMessages()
	c.viewport.GotoBottom()
}

// ClearPartial drops the live reply without adding a message, for a turn that
// ended in an error or was cancelled.
func (c *ChatModel) ClearPartial() { c.SetPartial("") }

func (c *ChatModel) AddAgentMessage(content, provider, model string, tokens int) {
	msg := Message{
		Role:     "agent",
		Content:  content,
		Provider: provider,
		Model:    model,
		Tokens:   tokens,
		Time:     time.Now(),
	}
	c.partial = ""
	c.messages = append(c.messages, msg)
	c.renderMessages()
	c.viewport.GotoBottom()
}

func (c *ChatModel) Clear() {
	c.messages = nil
	c.partial = ""
	c.viewport.SetContent("")
}

func (c *ChatModel) renderMessages() {
	var sb strings.Builder
	sb.WriteString("\n")

	for _, msg := range c.messages {
		switch msg.Role {
		case "user":
			c.renderBubble(&sb, msg, UserBubbleStyle, "▶ You", "")
		case "agent":
			meta := formatMetadata(msg.Provider, msg.Model, msg.Tokens)
			c.renderBubble(&sb, msg, AgentBubbleStyle, "◆ BOI", meta)
		case "system":
			sb.WriteString(fmt.Sprintf("\n%s %s\n", SystemStyle.Render("•"), DimStyle.Render(msg.Content)))
		case "error":
			sb.WriteString(fmt.Sprintf("\n%s %s\n", ErrorStyle.Render("✗"), ErrorStyle.Render(msg.Content)))
		}
	}

	if c.partial != "" {
		c.renderBubble(&sb, Message{Role: "agent", Content: c.partial, Time: time.Now()}, AgentBubbleStyle, "◆ BOI", "writing…")
	}

	c.viewport.SetContent(sb.String())
}

// minBubbleWidth keeps a box readable on a narrow terminal.
const minBubbleWidth = 20

// assumedTermWidth is used until the first WindowSizeMsg arrives. Without it
// c.width is 0, the box width goes negative and rendering panics.
const assumedTermWidth = 80

// bubbleWidth returns the inner width of a box holding content of the given
// display width, never wider than the terminal and never negative.
func (c *ChatModel) bubbleWidth(contentWidth int) int {
	terminalWidth := c.width
	if terminalWidth <= 0 {
		terminalWidth = assumedTermWidth
	}
	limit := terminalWidth - 6
	if limit < minBubbleWidth {
		limit = minBubbleWidth
	}
	boxW := contentWidth + 4
	if boxW < minBubbleWidth {
		boxW = minBubbleWidth
	}
	if boxW > limit {
		boxW = limit
	}
	return boxW
}

func (c *ChatModel) renderBubble(sb *strings.Builder, msg Message, style bubbleStyle, header, meta string) {
	content := strings.TrimSpace(msg.Content)
	if content == "" {
		return
	}

	// Size the box to the content, measured in display columns: len() counts
	// UTF-8 bytes and would make the box several times too wide for Thai.
	widest := term.StringWidth(header)
	for _, line := range strings.Split(content, "\n") {
		if w := term.StringWidth(line); w > widest {
			widest = w
		}
	}
	if meta != "" {
		if headerWidth := term.StringWidth(header + " · " + meta); headerWidth > widest {
			widest = headerWidth
		}
	}
	boxW := c.bubbleWidth(widest)
	// One column for the space after the left border, one so the text never
	// touches the right border. Without this the line was written at full
	// length, the terminal wrapped it, and the border landed mid-row.
	inner := boxW - 2
	if inner < 1 {
		inner = 1
	}

	// Header: the name always fits, the model and the clock only when there is
	// room for them.
	headerLine := style.header.Render(header)
	used := term.StringWidth(header)
	if meta != "" && used+3+term.StringWidth(meta) <= inner {
		headerLine += DimStyle.Render(" · ") + style.meta.Render(meta)
		used += 3 + term.StringWidth(meta)
	}
	if timeStr := msg.Time.Format("15:04"); used+2+len(timeStr) <= inner {
		headerLine += "  " + timeStampStyle.Render(timeStr)
		used += 2 + len(timeStr)
	}

	sb.WriteString("\n")
	sb.WriteString(style.topLeft)
	sb.WriteString(headerLine)
	sb.WriteString(strings.Repeat(style.horiz, max(1, boxW-used-2)))
	sb.WriteString(style.topRight)
	sb.WriteString("\n")

	// Empty line after header
	sb.WriteString(style.vert + strings.Repeat(" ", boxW) + style.vert + "\n")

	// Content lines, wrapped so every row is exactly as wide as the box.
	for _, line := range term.Wrap(content, inner) {
		sb.WriteString(style.vert + " " + line)
		sb.WriteString(strings.Repeat(" ", max(0, boxW-term.StringWidth(line)-1)) + style.vert + "\n")
	}

	// Empty line before bottom
	sb.WriteString(style.vert + strings.Repeat(" ", boxW) + style.vert + "\n")

	// Bottom
	sb.WriteString(style.botLeft)
	// botLeft/botRight already include the "─" connector chars, so repeat
	// boxW-2 here to keep the bottom edge the same width as the header and
	// content lines (boxW+2). Without this the box is 2 columns wider at the
	// bottom — a broken shape.
	sb.WriteString(strings.Repeat(style.horiz, max(1, boxW-2)))
	sb.WriteString(style.botRight)
	sb.WriteString("\n")
}

func formatMetadata(provider, model string, tokens int) string {
	if provider == "" && model == "" && tokens == 0 {
		return ""
	}
	parts := []string{}
	if provider != "" {
		parts = append(parts, provider)
	}
	if model != "" {
		parts = append(parts, model)
	}
	if tokens > 0 {
		parts = append(parts, fmt.Sprintf("%d tok", tokens))
	}
	return strings.Join(parts, " · ")
}

func (c *ChatModel) SetSize(w, h int) {
	changed := c.width != w
	c.width = w
	c.height = h
	c.viewport.Width = w - 4
	c.viewport.Height = max(h, 1)
	if changed {
		// Boxes are drawn to a fixed width, so a resize has to redraw them.
		// Without this they kept the width they were built at and the viewport
		// cut their right border away.
		atBottom := c.viewport.AtBottom()
		c.renderMessages()
		if atBottom {
			c.viewport.GotoBottom()
		}
	}
}

func (c *ChatModel) Update(msg tea.Msg) (ChatModel, tea.Cmd) {
	var cmd tea.Cmd
	c.viewport, cmd = c.viewport.Update(msg)
	return *c, cmd
}

func (c *ChatModel) View() string {
	return c.viewport.View()
}
