package agent

import (
	"context"
	"fmt"
	"strings"

	"github.com/boi-family/boi-cli/internal/block/port"
)

// ToolLookup resolves a registered Tool by name. *Broker satisfies it.
type ToolLookup interface {
	Tool(name string) (port.Tool, bool)
}

// RuntimeVerifier verifies host observations. It never treats a Model claim as
// proof that a Tool action or side effect occurred. Tool-specific checks are
// delegated to Tools that implement port.VerifiableTool.
type RuntimeVerifier struct{ Tools ToolLookup }

func (v RuntimeVerifier) Verify(ctx context.Context, input VerificationInput) (Verification, error) {
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
	var tool port.Tool
	if v.Tools != nil {
		tool, _ = v.Tools.Tool(call.Tool)
	}
	if verifiable, ok := tool.(port.VerifiableTool); ok {
		proof, err := verifiable.Verify(ctx, call.Arguments, port.ToolOutput{Output: result.Output, ChangedPaths: result.ChangedPaths})
		if err != nil {
			return Verification{Passed: false, Reason: err.Error()}, nil
		}
		return Verification{Passed: true, Evidence: append(evidence, proof)}, nil
	}
	if strings.TrimSpace(result.Output) == "" {
		return Verification{Passed: false, Reason: fmt.Sprintf("Tool %s returned no observable output", call.Tool)}, nil
	}
	return Verification{Passed: true, Evidence: evidence}, nil
}
