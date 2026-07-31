package agent

import "strings"

type Reviewer struct{}

func NewReviewer() *Reviewer {
	return &Reviewer{}
}

func (r *Reviewer) Review(response string) *ReviewResult {
	res := &ReviewResult{Passed: true}

	if len(strings.TrimSpace(response)) < 5 {
		res.Passed = false
		res.Reason = "response too short"
		res.Suggestion = "provide more detail"
	}

	if strings.Contains(strings.ToLower(response), "i don't know") && !strings.Contains(strings.ToLower(response), "let me") {
		res.Passed = false
		res.Reason = "agent gave up without trying"
		res.Suggestion = "try searching or reading files first"
	}

	return res
}

type ReviewResult struct {
	Passed     bool
	Reason     string
	Suggestion string
}
