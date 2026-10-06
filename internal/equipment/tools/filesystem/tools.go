package filesystem

import (
	"bytes"
	"context"
	"encoding/json"
	"fmt"
	"os"
	"strings"
	"time"

	"github.com/boi-family/boi-cli/internal/block/port"
	"github.com/boi-family/boi-cli/internal/runtime/workspace"
)

// Tools returns the workspace Tools bound to one workspace boundary.
func Tools(sandbox *workspace.Sandbox) []port.Tool {
	reader := NewReader(sandbox)
	return []port.Tool{listTool{reader}, readTool{reader}, writeTool{sandbox}}
}

type listTool struct{ reader *Reader }

func (listTool) Spec() port.ToolSpec {
	return port.ToolSpec{Name: "workspace.list", Usage: "workspace.list(path)", Risk: port.RiskRead, Approval: port.ApprovalAuto, Timeout: 5 * time.Second}
}

func (listTool) Describe(arguments map[string]any) port.ToolPreview {
	path := port.OptionalString(arguments, "path")
	return port.ToolPreview{Target: path, Preview: "List workspace directory: " + path}
}

func (t listTool) Execute(_ context.Context, arguments map[string]any) (port.ToolOutput, error) {
	path, err := port.StringArgument(arguments, "path")
	if err != nil {
		return port.ToolOutput{}, err
	}
	listing, err := t.reader.List(path)
	if err != nil {
		return port.ToolOutput{}, err
	}
	data, _ := json.Marshal(listing)
	return port.ToolOutput{Output: string(data)}, nil
}

type readTool struct{ reader *Reader }

func (readTool) Spec() port.ToolSpec {
	return port.ToolSpec{Name: "workspace.read", Usage: "workspace.read(path)", Risk: port.RiskRead, Approval: port.ApprovalAuto, Timeout: 5 * time.Second}
}

func (readTool) Describe(arguments map[string]any) port.ToolPreview {
	path := port.OptionalString(arguments, "path")
	return port.ToolPreview{Target: path, Preview: "Read workspace file: " + path}
}

func (t readTool) Execute(_ context.Context, arguments map[string]any) (port.ToolOutput, error) {
	path, err := port.StringArgument(arguments, "path")
	if err != nil {
		return port.ToolOutput{}, err
	}
	read, err := t.reader.Read(path)
	if err != nil {
		return port.ToolOutput{}, err
	}
	data, _ := json.Marshal(read)
	return port.ToolOutput{Output: string(data)}, nil
}

type writeTool struct{ sandbox *workspace.Sandbox }

func (writeTool) Spec() port.ToolSpec {
	return port.ToolSpec{Name: "workspace.write", Usage: "workspace.write(path, content)", Risk: port.RiskChange, Approval: port.ApprovalConfirm, Timeout: 10 * time.Second}
}

func (writeTool) Describe(arguments map[string]any) port.ToolPreview {
	return port.ToolPreview{Target: port.OptionalString(arguments, "path"), Preview: port.OptionalString(arguments, "content")}
}

func (t writeTool) Execute(_ context.Context, arguments map[string]any) (port.ToolOutput, error) {
	path, err := port.StringArgument(arguments, "path")
	if err != nil {
		return port.ToolOutput{}, err
	}
	content, err := port.StringArgument(arguments, "content")
	if err != nil {
		return port.ToolOutput{}, err
	}
	resolved, err := t.sandbox.ResolveForWrite(path)
	if err != nil {
		return port.ToolOutput{}, err
	}
	if err := os.WriteFile(resolved, []byte(content), 0o600); err != nil {
		return port.ToolOutput{}, fmt.Errorf("write workspace file: %w", err)
	}
	return port.ToolOutput{Output: "wrote " + path, ChangedPaths: []string{path}}, nil
}

func (listTool) Verify(_ context.Context, _ map[string]any, output port.ToolOutput) (port.Evidence, error) {
	return structuredObservation("workspace.list", output)
}

func (readTool) Verify(_ context.Context, _ map[string]any, output port.ToolOutput) (port.Evidence, error) {
	return structuredObservation("workspace.read", output)
}

func structuredObservation(name string, output port.ToolOutput) (port.Evidence, error) {
	if strings.TrimSpace(output.Output) == "" || !json.Valid([]byte(output.Output)) {
		return port.Evidence{}, fmt.Errorf("workspace observation is not valid structured data")
	}
	return port.Evidence{Kind: "structured_observation", Summary: name + " returned valid JSON"}, nil
}

// Verify reads the file back and requires the exact requested bytes.
func (t writeTool) Verify(_ context.Context, arguments map[string]any, _ port.ToolOutput) (port.Evidence, error) {
	path, err := port.StringArgument(arguments, "path")
	if err != nil {
		return port.Evidence{}, fmt.Errorf("write path is unavailable")
	}
	want, ok := arguments["content"].(string)
	if !ok {
		return port.Evidence{}, fmt.Errorf("write content is unavailable")
	}
	resolved, err := t.sandbox.ResolveExisting(path)
	if err != nil {
		return port.Evidence{}, fmt.Errorf("written path cannot be resolved")
	}
	got, err := os.ReadFile(resolved)
	if err != nil {
		return port.Evidence{}, fmt.Errorf("written file cannot be read back")
	}
	if !bytes.Equal(got, []byte(want)) {
		return port.Evidence{}, fmt.Errorf("written content does not match requested content")
	}
	return port.Evidence{Kind: "workspace_readback", Summary: "written bytes match", Ref: path}, nil
}
