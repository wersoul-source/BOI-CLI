package port

import (
	"testing"
	"time"
)

func TestToolSpecValidate(t *testing.T) {
	ok := ToolSpec{Name: "x", Risk: RiskRead, Approval: ApprovalAuto, Timeout: time.Second}
	if err := ok.Validate(); err != nil {
		t.Fatal(err)
	}
	bad := []ToolSpec{
		{Risk: RiskRead, Approval: ApprovalAuto, Timeout: time.Second},
		{Name: "x", Risk: "nope", Approval: ApprovalAuto, Timeout: time.Second},
		{Name: "x", Risk: RiskRead, Approval: "nope", Timeout: time.Second},
		{Name: "x", Risk: RiskChange, Approval: ApprovalAuto, Timeout: time.Second},
		{Name: "x", Risk: RiskRead, Approval: ApprovalAuto},
	}
	for _, spec := range bad {
		if spec.Validate() == nil {
			t.Errorf("accepted unsafe spec %+v", spec)
		}
	}
}

func TestArguments(t *testing.T) {
	args := map[string]any{"path": "a", "blank": "  ", "num": 3}
	if v, err := StringArgument(args, "path"); err != nil || v != "a" {
		t.Fatal("path")
	}
	for _, name := range []string{"blank", "num", "missing"} {
		if _, err := StringArgument(args, name); err == nil {
			t.Errorf("%s accepted", name)
		}
	}
	if OptionalString(args, "num") != "" || OptionalString(args, "path") != "a" {
		t.Fatal("OptionalString")
	}
}
