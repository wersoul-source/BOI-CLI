package subagent

import (
	"context"
	"errors"

	"github.com/boi-family/boi-cli/internal/runtime/agent"
)

var ErrSubagentsDisabled = errors.New("subagents are disabled until the evaluation gate is explicitly accepted")

// Enabled stays false until the evaluation gate is explicitly accepted.
const Enabled = false

type Subagent struct{}

func NewSubagent() *Subagent {
	return &Subagent{}
}

func (s *Subagent) Delegate(ctx context.Context, task string, persona string) (*agent.AgentResult, error) {
	return nil, ErrSubagentsDisabled
}
