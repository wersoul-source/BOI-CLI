package app

import (
	"fmt"
	"os"
	"strconv"
	"strings"
	"time"

	"github.com/boi-family/boi-cli/internal/runtime/agent"
)

// Environment variables that change one task's limits. The defaults suit a
// hosted Provider; a local model on a small GPU answers slowly and may need a
// longer task timeout, and a long context window allows a larger token budget.
const (
	EnvTaskTimeout  = "BOI_TASK_TIMEOUT"   // a duration such as 10m
	EnvMaxTokens    = "BOI_MAX_TOKENS"     // input plus output tokens of the whole task
	EnvMaxToolCalls = "BOI_MAX_TOOL_CALLS" // Tool calls of the whole task
	EnvMaxSteps     = "BOI_MAX_STEPS"      // decisions of the whole task
)

// LimitsFromEnv returns the engine limits with any BOI_* override applied. An
// invalid or out-of-range value is an error, never a silent default: the user
// asked for a specific limit.
func LimitsFromEnv() (agent.EngineLimits, error) {
	limits := agent.DefaultEngineLimits()
	if value, ok := lookupLimit(EnvTaskTimeout); ok {
		timeout, err := time.ParseDuration(value)
		if err != nil || timeout < 10*time.Second || timeout > 2*time.Hour {
			return limits, fmt.Errorf("%s=%q must be a duration between 10s and 2h, such as 10m", EnvTaskTimeout, value)
		}
		limits.Timeout = timeout
	}
	for _, item := range []struct {
		name     string
		target   *int
		min, max int
	}{
		{EnvMaxTokens, &limits.MaxTokens, 1_000, 4_000_000},
		{EnvMaxToolCalls, &limits.MaxToolCalls, 1, 200},
		{EnvMaxSteps, &limits.MaxSteps, 1, 200},
	} {
		value, ok := lookupLimit(item.name)
		if !ok {
			continue
		}
		number, err := strconv.Atoi(value)
		if err != nil || number < item.min || number > item.max {
			return limits, fmt.Errorf("%s=%q must be a whole number between %d and %d", item.name, value, item.min, item.max)
		}
		*item.target = number
	}
	return limits, nil
}

func lookupLimit(name string) (string, bool) {
	value := strings.TrimSpace(os.Getenv(name))
	return value, value != ""
}
