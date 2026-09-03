package cli

import (
	"bytes"
	"context"
	"encoding/json"
	"errors"
	"strings"
	"testing"
	"time"

	"github.com/boi-family/boi-cli/internal/agent"
)

func TestResolveAskQueryContract(t *testing.T) {
	t.Run("argv wins without reading stdin", func(t *testing.T) {
		query, err := resolveAskQuery([]string{"inspect", "repository"}, strings.NewReader("ignored"))
		if err != nil || query != "inspect repository" {
			t.Fatalf("query=%q err=%v", query, err)
		}
	})
	t.Run("UTF-8 stdin", func(t *testing.T) {
		query, err := resolveAskQuery(nil, strings.NewReader("  ตรวจระบบ\n"))
		if err != nil || query != "ตรวจระบบ" {
			t.Fatalf("query=%q err=%v", query, err)
		}
	})
	t.Run("empty stdin", func(t *testing.T) {
		_, err := resolveAskQuery(nil, strings.NewReader(" \n"))
		if ExitCode(err) != ExitInvalidInput {
			t.Fatalf("exit=%d err=%v", ExitCode(err), err)
		}
	})
	t.Run("bounded stdin", func(t *testing.T) {
		_, err := resolveAskQuery(nil, strings.NewReader(strings.Repeat("x", maxAutomationInputBytes+1)))
		if ExitCode(err) != ExitInvalidInput {
			t.Fatalf("exit=%d err=%v", ExitCode(err), err)
		}
	})
	t.Run("invalid UTF-8", func(t *testing.T) {
		_, err := resolveAskQuery(nil, bytes.NewReader([]byte{0xff, 0xfe}))
		if ExitCode(err) != ExitInvalidInput {
			t.Fatalf("exit=%d err=%v", ExitCode(err), err)
		}
	})
}

func TestIdempotencyKeyContract(t *testing.T) {
	for _, valid := range []string{"", "run-001", "tenant:a_task.2"} {
		if err := validateIdempotencyKey(valid); err != nil {
			t.Fatalf("valid key %q: %v", valid, err)
		}
	}
	for _, invalid := range []string{"has space", "slash/key", strings.Repeat("x", 129)} {
		if ExitCode(validateIdempotencyKey(invalid)) != ExitInvalidInput {
			t.Fatalf("invalid key accepted: %q", invalid)
		}
	}
}

func TestAutomationJSONSuccessIsOneVersionedObject(t *testing.T) {
	result := &agent.AgentResult{
		TaskID: "task-1", Response: "สำเร็จ", StopReason: agent.StopCompleted,
		Provider: "test", Model: "model", Manifest: "agent-folder/output/task-1/manifest.json",
		Artifacts:          []agent.ArtifactReference{{Path: "agent-folder/output/task-1/report.txt", SHA256: "abc", Bytes: 3}},
		IdempotencyKeyHash: "sha256:123", Steps: 2, Duration: 1500 * time.Millisecond,
		Usage: agent.Usage{InputTokens: 4, OutputTokens: 5, ProviderCalls: 1, ToolCalls: 1},
	}
	var output bytes.Buffer
	if err := writeAutomationResult(&output, result, nil); err != nil {
		t.Fatal(err)
	}
	decoder := json.NewDecoder(&output)
	var envelope AutomationResult
	if err := decoder.Decode(&envelope); err != nil {
		t.Fatal(err)
	}
	if envelope.SchemaVersion != 1 || envelope.Status != "completed" || envelope.TaskID != "task-1" || envelope.Usage.DurationMS != 1500 || len(envelope.Artifacts) != 1 {
		t.Fatalf("envelope=%#v", envelope)
	}
	var trailing any
	if err := decoder.Decode(&trailing); err == nil {
		t.Fatalf("JSON stream contains a second value: %#v", trailing)
	}
}

func TestAutomationNeverReportsEmptyResultAsCompleted(t *testing.T) {
	var output bytes.Buffer
	err := writeAutomationResult(&output, nil, nil)
	if ExitCode(err) != ExitInternal || !ErrorReported(err) {
		t.Fatalf("exit=%d reported=%v err=%v", ExitCode(err), ErrorReported(err), err)
	}
	var envelope AutomationResult
	if decodeErr := json.Unmarshal(output.Bytes(), &envelope); decodeErr != nil {
		t.Fatal(decodeErr)
	}
	if envelope.Status != "internal" || envelope.Artifacts == nil || envelope.Error == nil {
		t.Fatalf("envelope=%#v", envelope)
	}
}

func TestAutomationExitCodeMapping(t *testing.T) {
	cases := []struct {
		stop agent.StopReason
		code int
	}{
		{agent.StopNeedsApproval, ExitDenied},
		{agent.StopRejected, ExitDenied},
		{agent.StopSafetyBlocked, ExitDenied},
		{agent.StopCancelled, ExitCancelled},
		{agent.StopTimeout, ExitCancelled},
		{agent.StopProviderFailed, ExitUnavailable},
		{agent.StopVerificationFailed, ExitVerificationFailed},
		{agent.StopInvalidState, ExitInternal},
	}
	for _, test := range cases {
		result := &agent.AgentResult{TaskID: "task", StopReason: test.stop, Error: "failed"}
		var output bytes.Buffer
		err := writeAutomationResult(&output, result, errors.New("run failed"))
		if ExitCode(err) != test.code || !ErrorReported(err) {
			t.Fatalf("stop=%s exit=%d reported=%v err=%v", test.stop, ExitCode(err), ErrorReported(err), err)
		}
		var envelope AutomationResult
		if decodeErr := json.Unmarshal(output.Bytes(), &envelope); decodeErr != nil || envelope.Error == nil {
			t.Fatalf("stop=%s envelope=%#v decode=%v", test.stop, envelope, decodeErr)
		}
	}
	if ExitCode(context.Canceled) != ExitCancelled || ExitCode(errors.New("unknown")) != ExitInternal {
		t.Fatal("fallback exit mapping changed")
	}
}
