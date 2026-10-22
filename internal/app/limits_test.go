package app

import (
	"strings"
	"testing"
	"time"

	"github.com/boi-family/boi-cli/internal/runtime/agent"
)

func clearLimitEnv(t *testing.T) {
	t.Helper()
	for _, name := range []string{EnvTaskTimeout, EnvMaxTokens, EnvMaxToolCalls, EnvMaxSteps} {
		t.Setenv(name, "")
	}
}

func TestLimitsDefaultWithoutOverrides(t *testing.T) {
	clearLimitEnv(t)
	limits, err := LimitsFromEnv()
	if err != nil || limits != agent.DefaultEngineLimits() {
		t.Fatalf("limits = %+v, %v", limits, err)
	}
}

func TestLimitsCanBeRaisedForASlowLocalModel(t *testing.T) {
	clearLimitEnv(t)
	t.Setenv(EnvTaskTimeout, "15m")
	t.Setenv(EnvMaxTokens, "200000")
	t.Setenv(EnvMaxToolCalls, "30")
	t.Setenv(EnvMaxSteps, "40")
	limits, err := LimitsFromEnv()
	if err != nil {
		t.Fatal(err)
	}
	if limits.Timeout != 15*time.Minute || limits.MaxTokens != 200000 || limits.MaxToolCalls != 30 || limits.MaxSteps != 40 {
		t.Fatalf("limits = %+v", limits)
	}
	if limits.MaxRecoveries != agent.DefaultEngineLimits().MaxRecoveries {
		t.Fatal("recoveries are not configurable and must keep their default")
	}
}

func TestInvalidLimitIsAnErrorNotADefault(t *testing.T) {
	cases := map[string]string{
		EnvTaskTimeout:  "soon",
		EnvMaxTokens:    "lots",
		EnvMaxToolCalls: "0",
		EnvMaxSteps:     "5000",
	}
	for name, value := range cases {
		clearLimitEnv(t)
		t.Setenv(name, value)
		if _, err := LimitsFromEnv(); err == nil || !strings.Contains(err.Error(), name) {
			t.Errorf("%s=%q: error = %v", name, value, err)
		}
	}
	clearLimitEnv(t)
	t.Setenv(EnvTaskTimeout, "1s")
	if _, err := LimitsFromEnv(); err == nil {
		t.Error("a timeout below 10s must be refused")
	}
}

func TestAgentUsesTheLimitsFromTheEnvironment(t *testing.T) {
	clearLimitEnv(t)
	t.Setenv(EnvTaskTimeout, "bad")
	runtime := reportRuntime(t, nil)
	if _, err := runtime.BuildAgent(nil); err == nil || !strings.Contains(err.Error(), EnvTaskTimeout) {
		t.Fatalf("an invalid limit must stop start-up with its name: %v", err)
	}
}
