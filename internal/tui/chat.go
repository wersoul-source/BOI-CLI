package tui

import (
	"fmt"
	"strings"

	"github.com/charmbracelet/bubbles/viewport"
	tea "github.com/charmbracelet/bubbletea"
)

type Message struct {
	Role    string
	Content string
}

type ChatModel struct {
	viewport viewport.Model
	messages []Message
	width    int
	height   int
}

func NewChat() ChatModel {
	vp := viewport.New(80, 20)
	vp.Style = ChatBorderStyle
	return ChatModel{viewport: vp}
}

func (c *ChatModel) AddMessage(role, content string) {
	c.messages = append(c.messages, Message{Role: role, Content: content})
	c.renderMessages()
	c.viewport.GotoBottom()
}

func (c *ChatModel) Clear() {
	c.messages = nil
	c.viewport.SetContent("")
}

func (c *ChatModel) renderMessages() {
	var sb strings.Builder
	for _, msg := range c.messages {
		switch msg.Role {
		case "user":
			sb.WriteString(fmt.Sprintf("\n%s %s\n", UserStyle.Render("▶ You:"), msg.Content))
		case "agent":
			sb.WriteString(fmt.Sprintf("\n%s %s\n", AgentStyle.Render("◆ BOI:"), msg.Content))
		case "system":
			sb.WriteString(fmt.Sprintf("\n%s %s\n", InfoStyle.Render("•"), msg.Content))
		case "error":
			sb.WriteString(fmt.Sprintf("\n%s %s\n", ErrorStyle.Render("✗"), msg.Content))
		}
	}
	c.viewport.SetContent(sb.String())
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
