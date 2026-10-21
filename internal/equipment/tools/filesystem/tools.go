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
	return []port.Tool{listTool{reader}, readTool{reader}, searchTool{reader}, editTool{sandbox}, writeTool{sandbox}}
}

type listTool struct{ reader *Reader }

func (listTool) Spec() port.ToolSpec {
	return port.ToolSpec{Name: "workspace.list", Usage: "workspace.list(path)", Description: `List the files and folders under a workspace path. Use "." for the root.`, Risk: port.RiskRead, Approval: port.ApprovalAuto, Timeout: 5 * time.Second}
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

// maxReadPaths bounds one batched read.
const maxReadPaths = 5

func (readTool) Spec() port.ToolSpec {
	return port.ToolSpec{
		Name:        "workspace.read",
		Usage:       "workspace.read(path | paths)",
		Description: fmt.Sprintf("Read workspace text files. Give path for one file, or paths (up to %d) to read several files in one call. Use this, not process.run, to read files.", maxReadPaths),
		Parameters: map[string]any{
			"type": "object",
			"properties": map[string]any{
				"path":  map[string]any{"type": "string", "description": "one workspace-relative file path"},
				"paths": map[string]any{"type": "array", "items": map[string]any{"type": "string"}, "description": "several workspace-relative file paths"},
			},
			"additionalProperties": false,
		},
		Risk: port.RiskRead, Approval: port.ApprovalAuto, Timeout: 5 * time.Second,
	}
}

func (readTool) Describe(arguments map[string]any) port.ToolPreview {
	paths, _ := readPaths(arguments)
	target := strings.Join(paths, ", ")
	return port.ToolPreview{Target: target, Preview: "Read workspace file: " + target}
}

// readPaths returns the requested paths: exactly one of path or paths.
func readPaths(arguments map[string]any) ([]string, error) {
	single := port.OptionalString(arguments, "path")
	list, _ := arguments["paths"].([]any)
	if (single == "") == (len(list) == 0) {
		return nil, fmt.Errorf("give either path or paths")
	}
	if single != "" {
		return []string{single}, nil
	}
	if len(list) > maxReadPaths {
		return nil, fmt.Errorf("at most %d paths per call", maxReadPaths)
	}
	paths := make([]string, 0, len(list))
	for _, item := range list {
		path, ok := item.(string)
		if !ok || strings.TrimSpace(path) == "" {
			return nil, fmt.Errorf("paths must be non-empty strings")
		}
		paths = append(paths, path)
	}
	return paths, nil
}

type readFailure struct {
	Path  string
	Error string
}

func (t readTool) Execute(_ context.Context, arguments map[string]any) (port.ToolOutput, error) {
	paths, err := readPaths(arguments)
	if err != nil {
		return port.ToolOutput{}, err
	}
	if port.OptionalString(arguments, "path") != "" {
		read, err := t.reader.Read(paths[0])
		if err != nil {
			return port.ToolOutput{}, err
		}
		data, _ := json.Marshal(read)
		return port.ToolOutput{Output: string(data)}, nil
	}
	// A batch reports each file on its own, so one missing file does not hide
	// the others; it fails only when no file could be read.
	results := make([]any, 0, len(paths))
	failures := 0
	for _, path := range paths {
		read, err := t.reader.Read(path)
		if err != nil {
			failures++
			results = append(results, readFailure{Path: path, Error: err.Error()})
			continue
		}
		results = append(results, read)
	}
	data, _ := json.Marshal(results)
	if failures == len(paths) {
		return port.ToolOutput{}, fmt.Errorf("no file could be read: %s", string(data))
	}
	return port.ToolOutput{Output: string(data)}, nil
}

type searchTool struct{ reader *Reader }

func (searchTool) Spec() port.ToolSpec {
	return port.ToolSpec{
		Name:        "workspace.search",
		Usage:       "workspace.search(pattern, path)",
		Description: "Find the lines that contain a text, with file and line number, in one file or every text file under a folder. Use this before reading when you look for something specific in large or many files.",
		Parameters: map[string]any{
			"type": "object",
			"properties": map[string]any{
				"pattern": map[string]any{"type": "string", "description": "text to find, case-insensitive"},
				"path":    map[string]any{"type": "string", "description": "file or folder to search; defaults to the workspace root"},
			},
			"required":             []string{"pattern"},
			"additionalProperties": false,
		},
		Risk: port.RiskRead, Approval: port.ApprovalAuto, Timeout: 10 * time.Second,
	}
}

func (searchTool) Describe(arguments map[string]any) port.ToolPreview {
	path := port.OptionalString(arguments, "path")
	return port.ToolPreview{Target: path, Preview: fmt.Sprintf("Search %q in workspace %s", port.OptionalString(arguments, "pattern"), path)}
}

func (t searchTool) Execute(ctx context.Context, arguments map[string]any) (port.ToolOutput, error) {
	pattern, err := port.StringArgument(arguments, "pattern")
	if err != nil {
		return port.ToolOutput{}, err
	}
	result, err := t.reader.Search(ctx, pattern, port.OptionalString(arguments, "path"))
	if err != nil {
		return port.ToolOutput{}, err
	}
	data, _ := json.Marshal(result)
	return port.ToolOutput{Output: string(data)}, nil
}

type writeTool struct{ sandbox *workspace.Sandbox }

func (writeTool) Spec() port.ToolSpec {
	return port.ToolSpec{Name: "workspace.write", Usage: "workspace.write(path, content)", Description: "Create a new workspace file, or replace a whole file. To change part of an existing file use workspace.edit. Needs the user's approval.", Risk: port.RiskChange, Approval: port.ApprovalConfirm, Timeout: 10 * time.Second}
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
