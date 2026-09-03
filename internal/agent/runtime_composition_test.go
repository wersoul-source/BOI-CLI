package agent

import (
	"context"
	"errors"
	"os"
	"path/filepath"
	"reflect"
	"strings"
	"testing"
	"time"

	"github.com/boi-family/boi-cli/internal/persona"
	llm "github.com/boi-family/boi-cli/internal/provider"
	"github.com/boi-family/boi-cli/internal/skill"
	"github.com/boi-family/boi-cli/internal/workspace"
)

func TestPlannerProducesStableValidDependencyPlan(t *testing.T) {
	first := NewPlanner().Plan("inspect repository")
	second := NewPlanner().Plan("inspect repository")
	if err := first.Validate(); err != nil {
		t.Fatal(err)
	}
	if !reflect.DeepEqual(first, second) {
		t.Fatalf("Planner is not deterministic:\n%#v\n%#v", first, second)
	}
}

func TestRuntimeVerifierReadsBackWorkspaceWrite(t *testing.T) {
	root := t.TempDir()
	sandbox, err := workspace.NewSandbox(root)
	if err != nil {
		t.Fatal(err)
	}
	call := ToolCall{ID: "w1", Tool: "workspace.write", Purpose: "save", Arguments: map[string]any{"path": "out.txt", "content": "expected"}, Risk: RiskChange, Approval: ApprovalConfirm, Timeout: time.Second}
	if err := os.WriteFile(filepath.Join(root, "out.txt"), []byte("expected"), 0o600); err != nil {
		t.Fatal(err)
	}
	now := time.Now()
	result := ToolResult{CallID: "w1", Status: ToolSucceeded, Output: "wrote out.txt", StartedAt: now, FinishedAt: now.Add(time.Millisecond)}
	verification, err := (RuntimeVerifier{Sandbox: sandbox}).Verify(context.Background(), VerificationInput{ToolCall: &call, ToolResult: &result})
	if err != nil || !verification.Passed || len(verification.Evidence) < 2 {
		t.Fatalf("valid write not verified: %#v %v", verification, err)
	}
	if err := os.WriteFile(filepath.Join(root, "out.txt"), []byte("tampered"), 0o600); err != nil {
		t.Fatal(err)
	}
	verification, _ = (RuntimeVerifier{Sandbox: sandbox}).Verify(context.Background(), VerificationInput{ToolCall: &call, ToolResult: &result})
	if verification.Passed {
		t.Fatal("Model/Tool success claim passed despite failed read-back")
	}
}

func TestBoundedRecovererOnlyRetriesSafeClasses(t *testing.T) {
	recoverer := BoundedRecoverer{}
	transient := &llm.ProviderError{Class: llm.ErrorTransient, Message: "temporary"}
	if !recoverer.Recover(context.Background(), Failure{Phase: PhaseDecide, Err: transient, Attempt: 1}).Retry {
		t.Fatal("transient decision should retry")
	}
	auth := &llm.ProviderError{Class: llm.ErrorAuth, Message: "denied"}
	if recoverer.Recover(context.Background(), Failure{Phase: PhaseDecide, Err: auth, Attempt: 1}).Retry {
		t.Fatal("auth failure must not retry")
	}
	write := readCall()
	write.Risk = RiskChange
	if recoverer.Recover(context.Background(), Failure{Phase: PhaseAct, Err: errors.New("failed"), ToolCall: write, Attempt: 1}).Retry {
		t.Fatal("change action must not retry automatically")
	}
}

func TestServiceLoadsOnlyActiveSkillAsUntrustedContext(t *testing.T) {
	provider := &sequenceProvider{responses: []string{`<boi-skill>{"name":"git-helper"}</boi-skill>`, "used active instructions"}}
	service := NewService(persona.DefaultPersona(), llm.NewRouter([]llm.Provider{provider}), nil, nil)
	service.SetSkills([]*skill.Skill{{Name: "git-helper", Description: "Git helper", Prompt: "inspect status"}})
	result, err := service.Run(context.Background(), "use git helper")
	if err != nil {
		t.Fatal(err)
	}
	if result.Response != "used active instructions" || len(result.Trace) != 2 || result.Trace[0].Action != "skill:git-helper" {
		t.Fatalf("unexpected Skill flow: %#v", result)
	}
	if result.Usage.ToolCalls != 0 {
		t.Fatal("Skill Call incorrectly gained Tool authority")
	}
}

func TestServiceRejectsInactiveSkill(t *testing.T) {
	provider := &sequenceProvider{responses: []string{`<boi-skill>{"name":"not-active"}</boi-skill>`}}
	service := NewService(persona.DefaultPersona(), llm.NewRouter([]llm.Provider{provider}), nil, nil)
	service.SetSkills([]*skill.Skill{{Name: "git-helper", Description: "Git helper", Prompt: "inspect status"}})
	_, err := service.Run(context.Background(), "use an inactive skill")
	if err == nil || !strings.Contains(err.Error(), "Skill is not active for this task") {
		t.Fatalf("inactive Skill was not rejected deterministically: %v", err)
	}
}
