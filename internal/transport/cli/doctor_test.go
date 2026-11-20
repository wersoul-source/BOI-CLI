package cli

import (
	"strings"
	"testing"
)

// doctor reports failure through its exit code, so a script can rely on it.
// It used to exit 0 whatever it found (audit M10).
func TestDoctorVerdict(t *testing.T) {
	healthy := []checkResult{{label: "Go", ok: true}, {label: "Workspace", ok: true}, {label: "Config", ok: true}}
	if err := doctorVerdict(healthy); err != nil {
		t.Fatalf("healthy workspace: %v", err)
	}
	// A workspace still being set up is not a fault.
	unfinished := append([]checkResult{}, healthy...)
	unfinished = append(unfinished, checkResult{label: "Agent"}, checkResult{label: "Skills"}, checkResult{label: "Memory"}, checkResult{label: "Providers", warn: true})
	if err := doctorVerdict(unfinished); err != nil {
		t.Fatalf("advisory checks must not fail the command: %v", err)
	}
	broken := []checkResult{{label: "Go", ok: true}, {label: "Workspace"}, {label: "Config"}}
	err := doctorVerdict(broken)
	if err == nil || ExitCode(err) == ExitCompleted {
		t.Fatalf("a missing workspace and config must fail the command: %v", err)
	}
	if !strings.Contains(err.Error(), "Workspace") || !strings.Contains(err.Error(), "Config") {
		t.Fatalf("the error does not name what failed: %v", err)
	}
}

func TestDoctorFailsOutsideAWorkspace(t *testing.T) {
	t.Chdir(t.TempDir())
	if err := doctorCmd.RunE(doctorCmd, nil); err == nil || ExitCode(err) == ExitCompleted {
		t.Fatalf("doctor in an empty directory returned %v", err)
	}
}
