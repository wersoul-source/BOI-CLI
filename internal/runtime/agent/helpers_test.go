package agent

import (
	"context"
	"testing"
	"time"

	"github.com/boi-family/boi-cli/internal/block/port"
	"github.com/boi-family/boi-cli/internal/equipment/tools/filesystem"
	"github.com/boi-family/boi-cli/internal/equipment/tools/process"
	"github.com/boi-family/boi-cli/internal/runtime/workspace"
)

// Runtime tests use the real built-in Equipment. Equipment depends only on
// block/port and runtime/workspace, so importing it here creates no cycle.
func builtinTools(sandbox *workspace.Sandbox) []port.Tool {
	return append(filesystem.Tools(sandbox), process.Tool(sandbox))
}

type brokerFixture struct {
	*Broker
	root string
}

func testBroker(t *testing.T) brokerFixture {
	t.Helper()
	sandbox, err := workspace.NewSandbox(t.TempDir())
	if err != nil {
		t.Fatal(err)
	}
	b := NewBroker()
	activateAll(t, b, builtinTools(sandbox))
	return brokerFixture{Broker: b, root: sandbox.Root()}
}

func activateAll(t *testing.T, b *Broker, tools []port.Tool) {
	t.Helper()
	var names []string
	for _, tool := range tools {
		if err := b.Register(tool); err != nil {
			t.Fatal(err)
		}
		names = append(names, tool.Spec().Name)
	}
	if err := b.SetActiveCapabilities(names); err != nil {
		t.Fatal(err)
	}
}

// withWorkspaceTools registers and activates the built-in Tools on a Service.
func withWorkspaceTools(t *testing.T, service *Service, sandbox *workspace.Sandbox) *Service {
	t.Helper()
	activateAll(t, service.broker, builtinTools(sandbox))
	return service
}

type fakeTool struct {
	name  string
	calls int
}

func (f *fakeTool) Spec() port.ToolSpec {
	return port.ToolSpec{Name: f.name, Usage: f.name + "()", Risk: port.RiskRead, Approval: port.ApprovalAuto, Timeout: time.Second}
}
func (f *fakeTool) Describe(map[string]any) port.ToolPreview { return port.ToolPreview{} }
func (f *fakeTool) Execute(context.Context, map[string]any) (port.ToolOutput, error) {
	f.calls++
	return port.ToolOutput{Output: "ok"}, nil
}
