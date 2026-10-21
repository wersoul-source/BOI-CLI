// Package subagent is the SubAgent Block (ADR-003): one Tool that hands a
// self-contained sub-task to a child Agent restricted to a subset of the
// parent's Tools. The Runtime enforces the restrictions; this Block only
// asks for them.
package subagent

import (
	"context"
	"encoding/json"
	"fmt"
	"strings"
	"time"

	"github.com/boi-family/boi-cli/internal/block/port"
	"github.com/boi-family/boi-cli/internal/runtime/agent"
)

// Delegator is the Runtime port a SubAgent needs; *agent.Service implements it.
type Delegator interface {
	DelegableSpecs(names []string) ([]port.ToolSpec, error)
	RunChild(ctx context.Context, task string, tools []string) (*agent.AgentResult, error)
}

// Tool is agent.delegate. It is bound to the parent Service after the
// Service exists, because the Service is built from the Tools.
type Tool struct {
	delegator Delegator
}

func NewTool() *Tool { return &Tool{} }

// Bind connects the Tool to the Service it delegates from.
func (t *Tool) Bind(delegator Delegator) { t.delegator = delegator }

func (t *Tool) Spec() port.ToolSpec {
	return port.ToolSpec{
		Name:        agent.DelegateTool,
		Usage:       "agent.delegate(task, tools)",
		Description: "Hand a self-contained sub-task to a helper agent that may use only the listed tools and returns its answer. Use it for a separable piece of work, not for one tool call.",
		Parameters: map[string]any{
			"type": "object",
			"properties": map[string]any{
				"task":  map[string]any{"type": "string", "description": "complete instructions for the helper; it sees nothing else"},
				"tools": map[string]any{"type": "array", "items": map[string]any{"type": "string"}, "description": "tool names the helper may use, e.g. workspace.read"},
			},
			"required":             []string{"task", "tools"},
			"additionalProperties": false,
		},
		// The ceiling; each call is classified from the Tools it grants.
		Risk:     port.RiskExternal,
		Approval: port.ApprovalConfirm,
		Timeout:  agent.ChildLimits.Timeout + 10*time.Second,
	}
}

// Classify gives a call the highest risk among the Tools it grants. It is
// automatic only when every granted Tool is an automatic read.
func (t *Tool) Classify(arguments map[string]any) (port.Risk, port.Approval, error) {
	if t.delegator == nil {
		return "", "", fmt.Errorf("SubAgent is not connected to a task")
	}
	specs, err := t.delegator.DelegableSpecs(toolNames(arguments))
	if err != nil {
		return "", "", err
	}
	risk, approval := port.RiskRead, port.ApprovalAuto
	for _, spec := range specs {
		if spec.Risk.Rank() > risk.Rank() {
			risk = spec.Risk
		}
		if spec.Approval != port.ApprovalAuto {
			approval = port.ApprovalConfirm
		}
	}
	return risk, approval, nil
}

func (t *Tool) Describe(arguments map[string]any) port.ToolPreview {
	tools := strings.Join(toolNames(arguments), ", ")
	return port.ToolPreview{Target: tools, Preview: fmt.Sprintf("Delegate to a helper with [%s]: %s", tools, port.OptionalString(arguments, "task"))}
}

type delegated struct {
	Response     string   `json:"response"`
	StopReason   string   `json:"stop_reason"`
	Steps        int      `json:"steps"`
	ToolCalls    int      `json:"tool_calls"`
	ChangedPaths []string `json:"changed_paths,omitempty"`
}

func (t *Tool) Execute(ctx context.Context, arguments map[string]any) (port.ToolOutput, error) {
	if t.delegator == nil {
		return port.ToolOutput{}, fmt.Errorf("SubAgent is not connected to a task")
	}
	task, err := port.StringArgument(arguments, "task")
	if err != nil {
		return port.ToolOutput{}, err
	}
	result, err := t.delegator.RunChild(ctx, task, toolNames(arguments))
	if result == nil {
		if err == nil {
			err = fmt.Errorf("helper returned no result")
		}
		return port.ToolOutput{}, err
	}
	var changed []string
	for _, step := range result.Trace {
		if step.ToolResult != nil {
			changed = append(changed, step.ToolResult.ChangedPaths...)
		}
	}
	report := delegated{Response: result.Response, StopReason: string(result.StopReason), Steps: result.Steps, ToolCalls: result.Usage.ToolCalls, ChangedPaths: changed}
	data, _ := json.Marshal(report)
	if result.StopReason != agent.StopCompleted {
		return port.ToolOutput{}, fmt.Errorf("helper stopped (%s): %s; partial result %s", result.StopReason, result.Error, data)
	}
	return port.ToolOutput{Output: string(data), ChangedPaths: changed}, nil
}

func toolNames(arguments map[string]any) []string {
	list, _ := arguments["tools"].([]any)
	names := make([]string, 0, len(list))
	for _, item := range list {
		if name, ok := item.(string); ok && strings.TrimSpace(name) != "" {
			names = append(names, strings.TrimSpace(name))
		}
	}
	return names
}
