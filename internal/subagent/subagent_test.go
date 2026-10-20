package subagent

import (
	"context"
	"errors"
	"testing"
)

func TestSubagentsRemainDisabled(t *testing.T) {
	if Enabled {
		t.Fatal("subagents unexpectedly enabled")
	}
	if _, err := NewSubagent().Delegate(context.Background(), "task", "persona"); !errors.Is(err, ErrSubagentsDisabled) {
		t.Fatalf("got %v", err)
	}
}
