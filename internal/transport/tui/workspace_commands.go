package tui

import (
	"fmt"
	"strings"

	"github.com/boi-family/boi-cli/internal/tool/filesystem"
	tea "github.com/charmbracelet/bubbletea"
)

type workspaceResponseMsg struct {
	content string
	err     error
}

func isWorkspaceCommand(input string) bool {
	lower := strings.ToLower(strings.TrimSpace(input))
	return lower == "/workspace" || lower == "/ls" ||
		strings.HasPrefix(lower, "/ls ") || lower == "/read" ||
		strings.HasPrefix(lower, "/read ")
}

func (m *Model) callWorkspaceCmd(input string) tea.Cmd {
	return func() tea.Msg {
		if m.workspaceReader == nil {
			return workspaceResponseMsg{err: fmt.Errorf("workspace sandbox is not configured")}
		}

		trimmed := strings.TrimSpace(input)
		lower := strings.ToLower(trimmed)
		switch {
		case lower == "/workspace":
			return workspaceResponseMsg{content: fmt.Sprintf(
				"Workspace sandbox\nRoot: %s\nAccess: read-only",
				m.root,
			)}

		case lower == "/ls" || strings.HasPrefix(lower, "/ls "):
			path := strings.TrimSpace(trimmed[len("/ls"):])
			result, err := m.workspaceReader.List(path)
			if err != nil {
				return workspaceResponseMsg{err: err}
			}
			return workspaceResponseMsg{content: formatWorkspaceList(result)}

		case lower == "/read" || strings.HasPrefix(lower, "/read "):
			path := strings.TrimSpace(trimmed[len("/read"):])
			if path == "" {
				return workspaceResponseMsg{err: fmt.Errorf("usage: /read PATH")}
			}
			result, err := m.workspaceReader.Read(path)
			if err != nil {
				return workspaceResponseMsg{err: err}
			}
			return workspaceResponseMsg{content: formatWorkspaceRead(result)}
		}

		return workspaceResponseMsg{err: fmt.Errorf("unsupported workspace command")}
	}
}

func (m *Model) handleWorkspaceResponse(msg workspaceResponseMsg) {
	if msg.err != nil {
		m.chat.AddMessage("error", fmt.Sprintf("Workspace error: %v", msg.err))
		m.status.SetStatus("error")
		return
	}
	m.chat.AddMessage("system", msg.content)
	m.status.SetStatus("idle")
}

func formatWorkspaceList(result *filesystem.ListResult) string {
	var builder strings.Builder
	builder.WriteString(fmt.Sprintf("Workspace: %s\n", result.Path))
	for _, entry := range result.Entries {
		kind := "file"
		detail := fmt.Sprintf("%d B", entry.Size)
		if entry.IsDir {
			kind = "dir"
			detail = ""
		}
		builder.WriteString(fmt.Sprintf("[%s] %s", kind, entry.Name))
		if detail != "" {
			builder.WriteString("  " + detail)
		}
		builder.WriteByte('\n')
	}
	if len(result.Entries) == 0 {
		builder.WriteString("(empty)\n")
	}
	if result.Truncated {
		builder.WriteString("... list truncated at sandbox limit\n")
	}
	return strings.TrimRight(builder.String(), "\n")
}

func formatWorkspaceRead(result *filesystem.ReadResult) string {
	header := fmt.Sprintf("Workspace file: %s (%d B)", result.Path, result.Size)
	if result.Truncated {
		header += " [truncated]"
	}
	return header + "\n\n" + result.Content
}
