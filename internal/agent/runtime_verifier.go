package agent

import (
	"bytes"
	"context"
	"encoding/json"
	"fmt"
	"os"
	"strings"

	"github.com/boi-family/boi-cli/internal/workspace"
)

// RuntimeVerifier verifies host observations. It never treats a Model claim as
// proof that a Tool action or side effect occurred.
type RuntimeVerifier struct{ Sandbox *workspace.Sandbox }

func (v RuntimeVerifier) Verify(_ context.Context, input VerificationInput) (Verification, error) {
	if input.ToolResult == nil {
		if strings.TrimSpace(input.Response) == "" {
			return Verification{Passed: false, Reason: "response is empty"}, nil
		}
		return Verification{Passed: true, Reason: "direct response contains no host side-effect claim to verify"}, nil
	}
	result, call := input.ToolResult, input.ToolCall
	if call == nil || result.CallID != call.ID {
		return Verification{Passed: false, Reason: "Tool Result does not match Tool Call"}, nil
	}
	if err := result.Validate(); err != nil {
		return Verification{Passed: false, Reason: err.Error()}, nil
	}
	if result.Status != ToolSucceeded {
		return Verification{Passed: false, Reason: "Tool Result status is not succeeded"}, nil
	}
	evidence := []Evidence{{Kind: "tool_result", Summary: call.Tool + " succeeded", Ref: result.CallID}}
	switch call.Tool {
	case "workspace.write":
		if v.Sandbox == nil {
			return Verification{Passed: false, Reason: "workspace verifier is unavailable"}, nil
		}
		path, ok := stringArgument(call.Arguments, "path")
		if !ok {
			return Verification{Passed: false, Reason: "write path is unavailable"}, nil
		}
		want, ok := stringArgument(call.Arguments, "content")
		if !ok {
			return Verification{Passed: false, Reason: "write content is unavailable"}, nil
		}
		resolved, err := v.Sandbox.ResolveExisting(path)
		if err != nil {
			return Verification{Passed: false, Reason: "written path cannot be resolved"}, nil
		}
		got, err := os.ReadFile(resolved)
		if err != nil {
			return Verification{Passed: false, Reason: "written file cannot be read back"}, nil
		}
		if !bytes.Equal(got, []byte(want)) {
			return Verification{Passed: false, Reason: "written content does not match requested content"}, nil
		}
		evidence = append(evidence, Evidence{Kind: "workspace_readback", Summary: "written bytes match", Ref: path})
	case "workspace.read", "workspace.list":
		if strings.TrimSpace(result.Output) == "" || !json.Valid([]byte(result.Output)) {
			return Verification{Passed: false, Reason: "workspace observation is not valid structured data"}, nil
		}
	case "process.run":
		evidence = append(evidence, Evidence{Kind: "process_exit", Summary: "process executor reported success", Ref: result.CallID})
	default:
		if strings.TrimSpace(result.Output) == "" {
			return Verification{Passed: false, Reason: fmt.Sprintf("Tool %s returned no observable output", call.Tool)}, nil
		}
	}
	return Verification{Passed: true, Evidence: evidence}, nil
}
