package cli

import (
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"io"
	"os"
	"strings"
	"unicode/utf8"

	"github.com/boi-family/boi-cli/internal/agent"
)

const AutomationResultSchemaVersion = 1
const maxAutomationInputBytes = 1 << 20

const (
	ExitCompleted          = 0
	ExitInternal           = 1
	ExitInvalidInput       = 2
	ExitDenied             = 3
	ExitCancelled          = 4
	ExitUnavailable        = 5
	ExitVerificationFailed = 6
)

type CommandError struct {
	Code     int
	Class    string
	Message  string
	Cause    error
	Reported bool
}

func (e *CommandError) Error() string {
	if e == nil {
		return ""
	}
	if e.Message != "" {
		return e.Message
	}
	if e.Cause != nil {
		return e.Cause.Error()
	}
	return e.Class
}

func (e *CommandError) Unwrap() error { return e.Cause }

func ExitCode(err error) int {
	if err == nil {
		return ExitCompleted
	}
	var commandErr *CommandError
	if errors.As(err, &commandErr) && commandErr.Code >= ExitInternal && commandErr.Code <= ExitVerificationFailed {
		return commandErr.Code
	}
	if errors.Is(err, context.Canceled) || errors.Is(err, context.DeadlineExceeded) {
		return ExitCancelled
	}
	return ExitInternal
}

func ErrorReported(err error) bool {
	var commandErr *CommandError
	return errors.As(err, &commandErr) && commandErr.Reported
}

type AutomationResult struct {
	SchemaVersion      int                       `json:"schema_version"`
	Status             string                    `json:"status"`
	TaskID             string                    `json:"task_id,omitempty"`
	Response           string                    `json:"response,omitempty"`
	StopReason         agent.StopReason          `json:"stop_reason,omitempty"`
	Provider           string                    `json:"provider,omitempty"`
	Model              string                    `json:"model,omitempty"`
	Usage              AutomationUsage           `json:"usage"`
	Manifest           string                    `json:"manifest,omitempty"`
	Artifacts          []agent.ArtifactReference `json:"artifacts"`
	IdempotencyKeyHash string                    `json:"idempotency_key_hash,omitempty"`
	Error              *AutomationFailure        `json:"error,omitempty"`
}

type AutomationUsage struct {
	Steps         int   `json:"steps"`
	InputTokens   int   `json:"input_tokens"`
	OutputTokens  int   `json:"output_tokens"`
	ProviderCalls int   `json:"provider_calls"`
	ToolCalls     int   `json:"tool_calls"`
	DurationMS    int64 `json:"duration_ms"`
}

type AutomationFailure struct {
	Class   string `json:"class"`
	Message string `json:"message"`
}

func writeAutomationResult(w io.Writer, result *agent.AgentResult, runErr error) error {
	envelope, code, class := automationEnvelope(result, runErr)
	encoder := json.NewEncoder(w)
	encoder.SetEscapeHTML(false)
	if err := encoder.Encode(envelope); err != nil {
		return &CommandError{Code: ExitInternal, Class: "internal", Message: "write JSON result", Cause: err}
	}
	if code == ExitCompleted {
		return nil
	}
	return &CommandError{Code: code, Class: class, Message: envelope.Error.Message, Cause: runErr, Reported: true}
}

func automationEnvelope(result *agent.AgentResult, runErr error) (AutomationResult, int, string) {
	code, class := classifyAutomationFailure(result, runErr)
	envelope := AutomationResult{SchemaVersion: AutomationResultSchemaVersion, Status: class, Artifacts: []agent.ArtifactReference{}}
	if code == ExitCompleted {
		envelope.Status = "completed"
	}
	if result != nil {
		envelope.TaskID = result.TaskID
		envelope.Response = result.Response
		envelope.StopReason = result.StopReason
		envelope.Provider = result.Provider
		envelope.Model = result.Model
		envelope.Manifest = result.Manifest
		envelope.Artifacts = append([]agent.ArtifactReference{}, result.Artifacts...)
		envelope.IdempotencyKeyHash = result.IdempotencyKeyHash
		envelope.Usage = AutomationUsage{Steps: result.Steps, InputTokens: result.Usage.InputTokens, OutputTokens: result.Usage.OutputTokens, ProviderCalls: result.Usage.ProviderCalls, ToolCalls: result.Usage.ToolCalls, DurationMS: result.Duration.Milliseconds()}
	}
	if code != ExitCompleted {
		message := "command failed"
		if result != nil && strings.TrimSpace(result.Error) != "" {
			message = result.Error
		} else if runErr != nil {
			message = runErr.Error()
		}
		envelope.Error = &AutomationFailure{Class: class, Message: message}
	}
	return envelope, code, class
}

func classifyAutomationFailure(result *agent.AgentResult, err error) (int, string) {
	if err == nil && result != nil && result.StopReason == agent.StopCompleted {
		return ExitCompleted, "completed"
	}
	var commandErr *CommandError
	if errors.As(err, &commandErr) {
		return commandErr.Code, commandErr.Class
	}
	if result != nil {
		switch result.StopReason {
		case agent.StopCompleted:
			return ExitCompleted, "completed"
		case agent.StopNeedsApproval, agent.StopRejected, agent.StopSafetyBlocked:
			return ExitDenied, "denied"
		case agent.StopCancelled, agent.StopTimeout:
			return ExitCancelled, "cancelled"
		case agent.StopProviderFailed:
			return ExitUnavailable, "unavailable"
		case agent.StopVerificationFailed:
			return ExitVerificationFailed, "verification_failed"
		}
	}
	if errors.Is(err, context.Canceled) || errors.Is(err, context.DeadlineExceeded) {
		return ExitCancelled, "cancelled"
	}
	return ExitInternal, "internal"
}

func resolveAskQuery(args []string, input io.Reader) (string, error) {
	if len(args) > 0 {
		query := strings.TrimSpace(strings.Join(args, " "))
		if query == "" {
			return "", invalidInput("query argument is empty")
		}
		return query, nil
	}
	if input == nil || readerIsTerminal(input) {
		return "", invalidInput("query is required as argv or piped stdin")
	}
	data, err := io.ReadAll(io.LimitReader(input, maxAutomationInputBytes+1))
	if err != nil {
		return "", &CommandError{Code: ExitInvalidInput, Class: "invalid_input", Message: "read query from stdin", Cause: err}
	}
	if len(data) > maxAutomationInputBytes {
		return "", invalidInput(fmt.Sprintf("stdin query exceeds %d bytes", maxAutomationInputBytes))
	}
	if !utf8.Valid(data) {
		return "", invalidInput("stdin query must be valid UTF-8")
	}
	query := strings.TrimSpace(string(data))
	if query == "" {
		return "", invalidInput("stdin query is empty")
	}
	return query, nil
}

func validateIdempotencyKey(key string) error {
	if key == "" {
		return nil
	}
	if len(key) > 128 {
		return invalidInput("idempotency key exceeds 128 characters")
	}
	for _, value := range key {
		if (value >= 'a' && value <= 'z') || (value >= 'A' && value <= 'Z') || (value >= '0' && value <= '9') || strings.ContainsRune("._:-", value) {
			continue
		}
		return invalidInput("idempotency key may contain only letters, numbers, '.', '_', ':', and '-'")
	}
	return nil
}

func readerIsTerminal(input io.Reader) bool {
	file, ok := input.(*os.File)
	if !ok {
		return false
	}
	info, err := file.Stat()
	return err == nil && info.Mode()&os.ModeCharDevice != 0
}

func invalidInput(message string) error {
	return &CommandError{Code: ExitInvalidInput, Class: "invalid_input", Message: message}
}
