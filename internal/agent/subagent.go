package agent

import (
	"context"
	"fmt"
)

type Subagent struct{}

func NewSubagent() *Subagent {
	return &Subagent{}
}

func (s *Subagent) Delegate(ctx context.Context, task string, persona string) (*AgentResult, error) {
	return &AgentResult{
		Response: fmt.Sprintf("[Subagent %s] Task delegated: %s (not yet implemented)", persona, task),
		Steps:    0,
	}, nil
}
