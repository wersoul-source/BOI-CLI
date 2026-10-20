package mcp

import (
	"context"
	"fmt"
	"strings"
	"time"

	"github.com/boi-family/boi-cli/internal/block/port"
)

// Invoker calls one tool on one MCP server. *Client satisfies it.
type Invoker interface {
	CallTool(ctx context.Context, server, tool string, arguments map[string]any) (string, error)
}

// Tools adapts discovered MCP tools into Runtime Tools named
// "mcp.<server>.<tool>". Every adapted Tool is external and approval-gated.
func Tools(server string, names []string, invoker Invoker) ([]port.Tool, error) {
	if strings.TrimSpace(server) == "" || invoker == nil {
		return nil, fmt.Errorf("MCP server and invoker are required")
	}
	tools := make([]port.Tool, 0, len(names))
	for _, name := range names {
		if strings.TrimSpace(name) == "" {
			return nil, fmt.Errorf("MCP tool name is empty")
		}
		tools = append(tools, externalTool{server: server, tool: name, invoker: invoker})
	}
	return tools, nil
}

type externalTool struct {
	server  string
	tool    string
	invoker Invoker
}

func (t externalTool) Spec() port.ToolSpec {
	name := "mcp." + t.server + "." + t.tool
	return port.ToolSpec{Name: name, Usage: name + "(server-defined arguments)", Risk: port.RiskExternal, Approval: port.ApprovalConfirm, Timeout: 30 * time.Second}
}

func (t externalTool) Describe(map[string]any) port.ToolPreview {
	return port.ToolPreview{Target: t.server, Preview: "Call MCP tool " + t.tool + " on server " + t.server}
}

func (t externalTool) Execute(ctx context.Context, arguments map[string]any) (port.ToolOutput, error) {
	output, err := t.invoker.CallTool(ctx, t.server, t.tool, arguments)
	if err != nil {
		return port.ToolOutput{}, fmt.Errorf("external capability mcp.%s.%s: %w", t.server, t.tool, err)
	}
	return port.ToolOutput{Output: output}, nil
}
