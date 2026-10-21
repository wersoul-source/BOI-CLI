package process

import (
	"context"
	"fmt"
	"strings"
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
	return port.ToolSpec{Name: "process.run", Usage: "process.run(command)", Description: "Run one shell command in the workspace. Needs the user's approval and is denied in non-interactive runs; never use it to read or list files.", Risk: port.RiskExecute, Approval: port.ApprovalConfirm, Timeout: 30 * time.Second}
}

func (t runTool) Describe(arguments map[string]any) port.ToolPreview {
	return port.ToolPreview{Target: t.root, Preview: port.OptionalString(arguments, "command")}
}

func (t runTool) Execute(ctx context.Context, arguments map[string]any) (port.ToolOutput, error) {
	command, err := port.StringArgument(arguments, "command")
	if err != nil {
		return port.ToolOutput{}, err
	}
	result, err := t.executor.RunResult(ctx, command)
	if err != nil {
		return port.ToolOutput{}, err
	}
	// A non-zero exit is reported, not raised: a failing test run is exactly
	// what a coding task needs to see, and must not end the task.
	output := fmt.Sprintf("exit status: %d\n--- stdout ---\n%s\n--- stderr ---\n%s", result.ExitCode, result.Stdout, result.Stderr)
	if result.Truncated {
		output += "\n(output truncated)"
	}
	return port.ToolOutput{Output: output}, nil
}

// Verify records that the command ran to completion. Its exit status is in
// the output for the model to judge; empty output is not a failure.
func (runTool) Verify(_ context.Context, _ map[string]any, output port.ToolOutput) (port.Evidence, error) {
	status, _, _ := strings.Cut(output.Output, "\n")
	return port.Evidence{Kind: "process_exit", Summary: "process ran to completion, " + status}, nil
}
