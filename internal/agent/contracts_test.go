package agent

import (
	"testing"
	"time"
)

func validToolCall() ToolCall {
	return ToolCall{
		ID:             "call_001",
		Tool:           "filesystem.write",
		Purpose:        "update agent timeout",
		Arguments:      map[string]any{"path": "internal/agent/service.go"},
		Target:         "internal/agent/service.go",
		ExpectedResult: "the timeout is updated",
		Risk:           RiskChange,
		Approval:       ApprovalConfirm,
		Timeout:        10 * time.Second,
	}
}

func TestToolCallValidation(t *testing.T) {
	if err := validToolCall().Validate(); err != nil {
		t.Fatalf("valid call rejected: %v", err)
	}

	tests := []struct {
		name   string
		mutate func(*ToolCall)
	}{
		{"missing id", func(c *ToolCall) { c.ID = "" }},
		{"missing tool", func(c *ToolCall) { c.Tool = "" }},
		{"missing purpose", func(c *ToolCall) { c.Purpose = "" }},
		{"missing timeout", func(c *ToolCall) { c.Timeout = 0 }},
		{"automatic write", func(c *ToolCall) { c.Approval = ApprovalAuto }},
		{"weak critical approval", func(c *ToolCall) {
			c.Risk = RiskCritical
			c.Approval = ApprovalConfirm
		}},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			call := validToolCall()
			tt.mutate(&call)
			if err := call.Validate(); err == nil {
				t.Fatal("expected validation error")
			}
		})
	}
}

func TestApprovalRequestRejectsExpiredOrMutatedState(t *testing.T) {
	now := time.Now()
	request, err := NewApprovalRequest("approval_001", validToolCall(), now, now.Add(time.Minute))
	if err != nil {
		t.Fatalf("create approval request: %v", err)
	}
	if err := request.Validate(now); err != nil {
		t.Fatalf("valid request rejected: %v", err)
	}

	expired := request
	expired.ExpiresAt = now
	if err := expired.Validate(now); err == nil {
		t.Fatal("expected expired request to fail")
	}

	decided := request
	decided.State = ApprovalApproved
	if err := decided.Validate(now); err == nil {
		t.Fatal("expected pre-decided request to fail")
	}

	mutated := request
	mutated.Call.Target = "different-file.go"
	if err := mutated.Validate(now); err == nil {
		t.Fatal("expected changed tool call to invalidate approval")
	}
}

func TestToolResultValidation(t *testing.T) {
	started := time.Now()
	result := ToolResult{
		CallID:     "call_001",
		Status:     ToolSucceeded,
		StartedAt:  started,
		FinishedAt: started.Add(time.Second),
	}
	if err := result.Validate(); err != nil {
		t.Fatalf("valid result rejected: %v", err)
	}

	result.FinishedAt = started.Add(-time.Second)
	if err := result.Validate(); err == nil {
		t.Fatal("expected invalid timestamps to fail")
	}
}

func TestUsageTotalTokens(t *testing.T) {
	usage := Usage{InputTokens: 21, OutputTokens: 34}
	if got := usage.TotalTokens(); got != 55 {
		t.Fatalf("total tokens = %d, want 55", got)
	}
}
