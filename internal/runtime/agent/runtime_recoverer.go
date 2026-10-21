package agent

import (
	"context"
	"errors"

	"github.com/boi-family/boi-cli/internal/block/port"
	llm "github.com/boi-family/boi-cli/internal/runtime/llm"
)

type BoundedRecoverer struct{}

func (BoundedRecoverer) Recover(_ context.Context, failure Failure) Recovery {
	if failure.Attempt > 2 {
		return Recovery{Retry: false, Reason: "recovery budget exhausted"}
	}
	switch failure.Phase {
	case PhaseDecide:
		class := llm.ClassifyError(failure.Err)
		if class == llm.ErrorTransient || class == llm.ErrorRateLimit || class == llm.ErrorUnavailable || class == llm.ErrorUnknown {
			return Recovery{Retry: true, Reason: "retry bounded decision"}
		}
	case PhaseAct:
		if failure.ToolCall != nil && failure.ToolCall.Risk == RiskRead {
			return Recovery{Retry: true, Reason: "retry idempotent read action"}
		}
		// A change or command that failed before doing anything (edit text
		// not found, command blocked) is shown to the model to correct.
		if errors.Is(failure.Err, port.ErrNoEffect) {
			return Recovery{Retry: true, Reason: "re-plan after a Tool call that changed nothing"}
		}
	case PhaseVerify:
		return Recovery{Retry: true, Reason: "re-plan after failed verification"}
	}
	return Recovery{Retry: false, Reason: "failure is not safely recoverable"}
}
