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
	width    int
	height   int
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

func (c *ChatModel) AddAgentMessage(content, provider, model string, tokens int) {
	msg := Message{
		Role:     "agent",
		Content:  content,
		Provider: provider,
		Model:    model,
		Tokens:   tokens,
		Time:     time.Now(),
	}
	c.messages = append(c.messages, msg)
	c.renderMessages()
	c.viewport.GotoBottom()
}

func (c *ChatModel) Clear() {
	c.messages = nil
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

	c.viewport.SetContent(sb.String())
}

func (c *ChatModel) renderBubble(sb *strings.Builder, msg Message, style bubbleStyle, header, meta string) {
	content := strings.TrimSpace(msg.Content)
	if content == "" {
		return
	}

	// Calculate max line width from content (display width, not byte length —
	// len() would count UTF-8 bytes and break the box shape for Thai text).
	lines := strings.Split(content, "\n")
	maxLine := 0
	for _, l := range lines {
		w := term.ThaiStringWidth(l)
		if w > maxLine {
			maxLine = w
		}
	}
	headerLen := term.ThaiStringWidth(header)
	if meta != "" {
		headerLen += term.ThaiStringWidth(" · ") + term.ThaiStringWidth(meta)
	}
	if headerLen > maxLine {
		maxLine = headerLen
	}
	boxW := maxLine + 4 // padding
	if boxW < 20 {
		boxW = 20
	}
	if boxW > c.width-6 {
		boxW = c.width - 6
	}

	// Header line
	headerLine := style.header.Render(header)
	if meta != "" {
		headerLine += DimStyle.Render(" · ") + style.meta.Render(meta)
	}
	timeStr := msg.Time.Format("15:04")
	headerLine += "  " + timeStampStyle.Render(timeStr)

	sb.WriteString("\n")
	sb.WriteString(style.topLeft)
	sb.WriteString(headerLine)
	sb.WriteString(strings.Repeat(style.horiz, max(1, boxW-term.ThaiStringWidth(headerLine)-2)))
	sb.WriteString(style.topRight)
	sb.WriteString("\n")

	// Empty line after header
	sb.WriteString(style.vert + strings.Repeat(" ", boxW) + style.vert + "\n")

	// Content lines
	for _, line := range lines {
		sb.WriteString(style.vert + " " + line)
		pad := boxW - term.ThaiStringWidth(line) - 1
		if pad < 0 {
			pad = 0
		}
		sb.WriteString(strings.Repeat(" ", pad) + style.vert + "\n")
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
	c.width = w
	c.height = h
	c.viewport.Width = w - 4
	c.viewport.Height = max(h, 1)
}

func (c *ChatModel) Update(msg tea.Msg) (ChatModel, tea.Cmd) {
	var cmd tea.Cmd
	c.viewport, cmd = c.viewport.Update(msg)
	return *c, cmd
}

func (c *ChatModel) View() string {
	return c.viewport.View()
}
