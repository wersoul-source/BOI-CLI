package process

import (
	"context"
	"time"

	"github.com/boi-family/boi-cli/internal/block/port"
	"github.com/boi-family/boi-cli/internal/runtime/workspace"
)

// Tool returns process.run bound to one workspace boundary.
func Tool(sandbox *workspace.Sandbox) port.Tool {
	return runTool{root: sandbox.Root(), executor: NewExecutor(WithWorkspace(sandbox))}
}

type runTool struct {
	root     string
	executor *Executor
}

func (runTool) Spec() port.ToolSpec {
	return port.ToolSpec{Name: "process.run", Usage: "process.run(command)", Risk: port.RiskExecute, Approval: port.ApprovalConfirm, Timeout: 30 * time.Second}
}

func (t runTool) Describe(arguments map[string]any) port.ToolPreview {
	return port.ToolPreview{Target: t.root, Preview: port.OptionalString(arguments, "command")}
}

func (t runTool) Execute(ctx context.Context, arguments map[string]any) (port.ToolOutput, error) {
	command, err := port.StringArgument(arguments, "command")
	if err != nil {
		return port.ToolOutput{}, err
	}
	output, err := t.executor.RunContext(ctx, command)
	if err != nil {
		return port.ToolOutput{}, err
	}
	return port.ToolOutput{Output: output}, nil
}

// Verify records that the executor reported a zero exit status. A process
// may legitimately print nothing, so empty output is not a failure.
func (runTool) Verify(context.Context, map[string]any, port.ToolOutput) (port.Evidence, error) {
	return port.Evidence{Kind: "process_exit", Summary: "process executor reported success"}, nil
}
