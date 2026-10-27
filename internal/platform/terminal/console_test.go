package terminal

import "testing"

// TestSetUTF8Console verifies the function doesn't panic (smoke test).
// Actual effect requires a real console — can't be tested in unit tests.
func TestSetUTF8Console(t *testing.T) {
	defer func() {
		if r := recover(); r != nil {
			t.Errorf("SetUTF8Console panicked: %v", r)
		}
	}()
	SetUTF8Console()
}

// TestThaiFontInstalled is a smoke test — should not panic on Windows.
func TestThaiFontInstalled(t *testing.T) {
	defer func() {
		if r := recover(); r != nil {
			t.Errorf("ThaiFontInstalled panicked: %v", r)
		}
	}()
	_ = ThaiFontInstalled()
}
