package tui

import (
	"fmt"
	"strings"
	"time"

	"github.com/boi-family/boi-cli/internal/agent"
)

const approvalPreviewLines = 5

// ApprovalModel renders one exact host authorization request. It contains no
// execution logic and cannot mutate the tool call it displays.
type ApprovalModel struct {
	request *agent.ApprovalRequest
	width   int
}

func NewApproval() ApprovalModel {
	return ApprovalModel{}
}

func (m *ApprovalModel) Open(request agent.ApprovalRequest, now time.Time) error {
	if err := request.Validate(now); err != nil {
		return err
	}
	m.request = &request
	return nil
}

func (m *ApprovalModel) Close() {
	m.request = nil
}

func (m *ApprovalModel) Active() bool {
	return m.request != nil
}

func (m *ApprovalModel) Request() *agent.ApprovalRequest {
	return m.request
}

func (m *ApprovalModel) SetWidth(width int) {
	m.width = width
}

func (m *ApprovalModel) Height() int {
	if !m.Active() {
		return 0
	}
	previewCount := len(previewLines(m.request.Call.Preview))
	if previewCount > approvalPreviewLines {
		previewCount = approvalPreviewLines
	}
	if previewCount == 0 {
		previewCount = 1
	}
	return 10 + previewCount
}

// Decide maps explicit keyboard input to an immutable approval decision.
// Enter is deliberately not accepted to prevent accidental authorization.
func (m *ApprovalModel) Decide(key string, now time.Time) (*agent.ApprovalDecision, bool) {
	if !m.Active() {
		return nil, false
	}
	decision := &agent.ApprovalDecision{
		RequestID: m.request.ID,
		DecidedAt: now,
	}
	if !m.request.ExpiresAt.IsZero() && !now.Before(m.request.ExpiresAt) {
		decision.State = agent.ApprovalExpired
		decision.Reason = "approval request expired"
		return decision, true
	}
	switch strings.ToLower(key) {
	case "a", "y":
		decision.State = agent.ApprovalApproved
	case "r", "n":
		decision.State = agent.ApprovalRejected
		decision.Reason = "rejected by user"
	case "esc", "ctrl+c":
		decision.State = agent.ApprovalCancelled
		decision.Reason = "task cancelled by user"
	default:
		return nil, false
	}
	return decision, true
}

func (m *ApprovalModel) View() string {
	if !m.Active() {
		return ""
	}
	call := m.request.Call
	target := call.Target
	if strings.TrimSpace(target) == "" {
		target = "(not specified)"
	}
	preview := previewLines(call.Preview)
	truncated := len(preview) > approvalPreviewLines
	if len(preview) == 0 {
		preview = []string{"(no preview supplied)"}
	}
	if truncated {
		preview = preview[:approvalPreviewLines]
		preview = append(preview, "... preview truncated")
	}

	var body strings.Builder
	body.WriteString(ApprovalTitleStyle.Render("! APPROVAL REQUIRED"))
	body.WriteString("\nPurpose: " + call.Purpose)
	body.WriteString("\nTool: " + call.Tool)
	body.WriteString("\nTarget: " + target)
	body.WriteString("\nRisk: " + strings.ToUpper(string(call.Risk)))
	body.WriteString("\nPreview:\n")
	for _, line := range preview {
		body.WriteString("  " + line + "\n")
	}
	body.WriteString("\n[A] Approve once   [R] Reject   [Esc] Cancel task")

	width := m.width
	if width < 20 {
		width = 20
	}
	return ApprovalPanelStyle.Width(width).Render(body.String())
}

func previewLines(preview string) []string {
	preview = strings.TrimSpace(preview)
	if preview == "" {
		return nil
	}
	return strings.Split(preview, "\n")
}

func approvalDecisionSummary(decision agent.ApprovalDecision) string {
	switch decision.State {
	case agent.ApprovalApproved:
		return fmt.Sprintf("Approval %s: approved once", decision.RequestID)
	case agent.ApprovalRejected:
		return fmt.Sprintf("Approval %s: rejected", decision.RequestID)
	case agent.ApprovalCancelled:
		return fmt.Sprintf("Approval %s: task cancelled", decision.RequestID)
	case agent.ApprovalExpired:
		return fmt.Sprintf("Approval %s: expired", decision.RequestID)
	default:
		return fmt.Sprintf("Approval %s: %s", decision.RequestID, decision.State)
	}
}
