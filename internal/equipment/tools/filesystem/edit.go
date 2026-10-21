package filesystem

import (
	"context"
	"fmt"
	"os"
	"strings"
	"time"
	"unicode/utf8"

	"github.com/boi-family/boi-cli/internal/block/port"
	"github.com/boi-family/boi-cli/internal/runtime/workspace"
)

// maxEditBytes bounds the file an edit loads into memory.
const maxEditBytes = 4 << 20

// editTool replaces one exact piece of text in an existing file. Models that
// rewrite a whole file to change one line drop or invent the lines they did
// not mean to touch; an edit leaves every other byte as it was.
type editTool struct{ sandbox *workspace.Sandbox }

func (editTool) Spec() port.ToolSpec {
	return port.ToolSpec{
		Name:        "workspace.edit",
		Usage:       "workspace.edit(path, old, new)",
		Description: "Change part of an existing workspace file: replace the exact text old with new. old must appear exactly once, so include enough surrounding lines to make it unique; copy it from workspace.read output. Use this instead of workspace.write to change an existing file. Needs the user's approval.",
		Parameters: map[string]any{
			"type": "object",
			"properties": map[string]any{
				"path": map[string]any{"type": "string", "description": "workspace-relative path of an existing file"},
				"old":  map[string]any{"type": "string", "description": "exact text to replace, including whitespace"},
				"new":  map[string]any{"type": "string", "description": "replacement text; empty deletes old"},
			},
			"required": []any{"path", "old", "new"},
		},
		Risk: port.RiskChange, Approval: port.ApprovalConfirm, Timeout: 10 * time.Second,
	}
}

func (editTool) Describe(arguments map[string]any) port.ToolPreview {
	preview := "- " + strings.ReplaceAll(port.OptionalString(arguments, "old"), "\n", "\n- ") + "\n+ " + strings.ReplaceAll(port.OptionalString(arguments, "new"), "\n", "\n+ ")
	return port.ToolPreview{Target: port.OptionalString(arguments, "path"), Preview: preview}
}

func (t editTool) Execute(_ context.Context, arguments map[string]any) (port.ToolOutput, error) {
	path := port.OptionalString(arguments, "path")
	resolved, updated, mode, line, err := t.plan(arguments)
	if err != nil {
		// Nothing was written: the model may correct old and try again.
		return port.ToolOutput{}, port.NoEffect(err)
	}
	if err := os.WriteFile(resolved, []byte(updated), mode); err != nil {
		return port.ToolOutput{}, fmt.Errorf("write %s: %w", path, err)
	}
	return port.ToolOutput{Output: fmt.Sprintf("edited %s at line %d", path, line), ChangedPaths: []string{path}}, nil
}

// plan computes the edited content without touching the file.
func (t editTool) plan(arguments map[string]any) (resolved, updated string, mode os.FileMode, line int, err error) {
	path, err := port.StringArgument(arguments, "path")
	if err != nil {
		return "", "", 0, 0, err
	}
	old, err := port.StringArgument(arguments, "old")
	if err != nil {
		return "", "", 0, 0, err
	}
	replacement, ok := arguments["new"].(string)
	if !ok {
		return "", "", 0, 0, fmt.Errorf("argument new must be a string")
	}
	if IsSecretFile(path) {
		return "", "", 0, 0, fmt.Errorf("refusing to edit a credential file: %s", path)
	}
	resolved, err = t.sandbox.ResolveExisting(path)
	if err != nil {
		return "", "", 0, 0, err
	}
	info, err := os.Stat(resolved)
	if err != nil {
		return "", "", 0, 0, fmt.Errorf("stat %s: %w", path, err)
	}
	if !info.Mode().IsRegular() {
		return "", "", 0, 0, fmt.Errorf("%s is not a regular file", path)
	}
	if info.Size() > maxEditBytes {
		return "", "", 0, 0, fmt.Errorf("%s is larger than %d bytes", path, maxEditBytes)
	}
	data, err := os.ReadFile(resolved)
	if err != nil {
		return "", "", 0, 0, fmt.Errorf("read %s: %w", path, err)
	}
	content := string(data)
	if !utf8.ValidString(content) {
		return "", "", 0, 0, fmt.Errorf("%s is not a text file", path)
	}
	// A model copies lines with \n; a file written on Windows uses \r\n.
	if !strings.Contains(content, old) && strings.Contains(content, "\r\n") {
		old, replacement = toCRLF(old), toCRLF(replacement)
	}
	switch count := strings.Count(content, old); count {
	case 1:
	case 0:
		return "", "", 0, 0, fmt.Errorf("old text not found in %s; read the file and copy the text exactly", path)
	default:
		return "", "", 0, 0, fmt.Errorf("old text appears %d times in %s; include more surrounding lines so it appears once", count, path)
	}
	line = strings.Count(content[:strings.Index(content, old)], "\n") + 1
	return resolved, strings.Replace(content, old, replacement, 1), info.Mode().Perm(), line, nil
}

// Verify reads the file back and requires the replacement to be present.
func (t editTool) Verify(_ context.Context, arguments map[string]any, _ port.ToolOutput) (port.Evidence, error) {
	path, err := port.StringArgument(arguments, "path")
	if err != nil {
		return port.Evidence{}, fmt.Errorf("edit path is unavailable")
	}
	resolved, err := t.sandbox.ResolveExisting(path)
	if err != nil {
		return port.Evidence{}, fmt.Errorf("edited path cannot be resolved")
	}
	data, err := os.ReadFile(resolved)
	if err != nil {
		return port.Evidence{}, fmt.Errorf("edited file cannot be read back")
	}
	replacement, _ := arguments["new"].(string)
	content := string(data)
	if !strings.Contains(content, replacement) && !strings.Contains(content, toCRLF(replacement)) {
		return port.Evidence{}, fmt.Errorf("edited file does not contain the replacement text")
	}
	return port.Evidence{Kind: "workspace_readback", Summary: "replacement text present", Ref: path}, nil
}

func toCRLF(text string) string {
	return strings.ReplaceAll(strings.ReplaceAll(text, "\r\n", "\n"), "\n", "\r\n")
}
