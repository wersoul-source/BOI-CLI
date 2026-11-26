package cli

import (
	"bytes"
	"context"
	"encoding/json"
	"errors"
	"strings"
	"testing"
	"time"

	"github.com/boi-family/boi-cli/internal/runtime/agent"
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

// A command error tells the person the cause, once. "Error: load providers"
// used to be the whole message when no Provider was configured (audit M3).
func TestCommandErrorCarriesItsCauseOnce(t *testing.T) {
	inner := unavailable("no Provider is configured; run 'boi setup' or 'boi quickstart'", errors.New("no valid providers found in environment (PSC_1..PSC_9)"))
	text := inner.Error()
	for _, want := range []string{"boi setup", "no valid providers found"} {
		if !strings.Contains(text, want) {
			t.Fatalf("message lacks %q: %s", want, text)
		}
	}
	// finishAsk wraps the error again with its own text as the message.
	outer := &CommandError{Code: ExitUnavailable, Class: "unavailable", Message: inner.Error(), Cause: inner}
	if outer.Error() != text {
		t.Fatalf("the cause was repeated:\n%s", outer.Error())
	}
	if got := (&CommandError{Message: "only a message"}).Error(); got != "only a message" {
		t.Fatalf("got %q", got)
	}
	if got := (&CommandError{Class: "internal", Cause: errors.New("boom")}).Error(); got != "boom" {
		t.Fatalf("got %q", got)
	}
}

// The notice and the misquotes reach a script that reads --json, and an
// answer with neither has no such field, so existing consumers see no change.
func TestAutomationJSONCarriesTheNoticeAndMisquotes(t *testing.T) {
	plain := &agent.AgentResult{TaskID: "t", Response: "ok", StopReason: agent.StopCompleted}
	flagged := &agent.AgentResult{TaskID: "t", Response: "It is ZOO-MARKER-public", StopReason: agent.StopCompleted,
		Notice:    "This answer may misquote what was read",
		Misquotes: []agent.Misquote{{Answer: "ZOO-MARKER-public", Observed: "ZOO-MARKER-42"}}}
	var out bytes.Buffer
	if err := writeAutomationResult(&out, plain, nil); err != nil {
		t.Fatal(err)
	}
	if strings.Contains(out.String(), "notice") || strings.Contains(out.String(), "misquotes") {
		t.Fatalf("an unflagged result grew new fields: %s", out.String())
	}
	out.Reset()
	if err := writeAutomationResult(&out, flagged, nil); err != nil {
		t.Fatal(err)
	}
	var envelope AutomationResult
	if err := json.Unmarshal(out.Bytes(), &envelope); err != nil {
		t.Fatal(err)
	}
	if envelope.SchemaVersion != 1 || envelope.Status != "completed" || !strings.Contains(envelope.Notice, "misquote") || len(envelope.Misquotes) != 1 || envelope.Misquotes[0].Observed != "ZOO-MARKER-42" {
		t.Fatalf("envelope = %#v\n%s", envelope, out.String())
	}
	if !strings.Contains(out.String(), `"answer":"ZOO-MARKER-public"`) || !strings.Contains(out.String(), `"observed":"ZOO-MARKER-42"`) {
		t.Fatalf("misquote field names changed: %s", out.String())
	}
	// A refused run carries its notice too.
	refused := &agent.AgentResult{TaskID: "t", StopReason: agent.StopNeedsApproval, Notice: "Every Tool call needs approval"}
	out.Reset()
	_ = writeAutomationResult(&out, refused, nil)
	if !strings.Contains(out.String(), "Every Tool call needs approval") {
		t.Fatalf("a refused run lost its notice: %s", out.String())
	}
}
