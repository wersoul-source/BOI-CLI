package agent

import (
	"context"
	"errors"
)

var ErrSubagentsDisabled = errors.New("subagents are disabled until the evaluation gate is explicitly accepted")

const SubagentsEnabled = false

type Subagent struct{}

func NewSubagent() *Subagent {
	return &Subagent{}
}

func (s *Subagent) Delegate(ctx context.Context, task string, persona string) (*AgentResult, error) {
	return nil, ErrSubagentsDisabled
}
