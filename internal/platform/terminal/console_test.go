package terminal

import (
	"testing"

	"github.com/mattn/go-runewidth"
)

// TestThaiWidth verifies that our ThaiStringWidth correctly counts
// Thai zero-width combining marks (EXP-A from thai-terminal-fix skill).
//
// Thai consonants = 1 cell
// Thai vowels/tone marks = 0 cells (zero-width combining marks)
//
// Note: go-runewidth v0.0.19 counts Thai marks as width=1 (BUG!).
// Our ThaiStringWidth corrects this.
func TestThaiWidth(t *testing.T) {
	tests := []struct {
		name     string
		input    string
		expected int
	}{
		{"consonant ก", "ก", 1},
		{"vowel-above ั", "ั", 0},     // U+0E31 MAI HAN-AKAT
		{"tone-à ่", "่", 0},          // U+0E48 MAI EK
		{"word ก่อน", "ก่อน", 3},      // ก + อ + น (marks don't add width)
		{"word สวัสดี", "สวัสดี", 4},  // ส + ว + ส + ด (2 marks = 0)
		{"sentence", "สวัสดีครับ", 7}, // 7 consonants (ส ว ส ด ค ร บ) + 3 marks (ั ี ั = 0)
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got := ThaiStringWidth(tt.input)
			if got != tt.expected {
				t.Errorf("ThaiStringWidth(%q) = %d, want %d", tt.input, got, tt.expected)
			}
		})
	}
}

// TestGoRunewidthBug documents the bug in go-runewidth v0.0.19
// that counts Thai marks as width=1 instead of 0.
func TestGoRunewidthBug(t *testing.T) {
	tests := []struct {
		rune rune
		desc string
	}{
		{0x0E31, "MAI HAN-AKAT (vowel above)"},
		{0x0E48, "MAI EK (tone mark)"},
	}

	for _, tt := range tests {
		t.Run(tt.desc, func(t *testing.T) {
			got := runewidth.RuneWidth(tt.rune)
			// This documents the bug: go-runewidth returns 1, should be 0
			if got != 0 {
				t.Logf("CONFIRMED BUG: runewidth.RuneWidth(%q U+%04X) = %d (should be 0) — %s",
					string(tt.rune), tt.rune, got, tt.desc)
			}
		})
	}
}

// TestThaiRuneWidth checks our custom zero-width detection.
func TestThaiRuneWidth(t *testing.T) {
	tests := []struct {
		rune   rune
		isZero bool
		desc   string
	}{
		{'ก', false, "KO KAI (consonant) = width 1"},
		{0x0E31, true, "MAI HAN-AKAT (vowel above) = width 0"},
		{0x0E48, true, "MAI EK (tone mark) = width 0"},
		{'ส', false, "SO SUA (consonant) = width 1"},
		{'◈', false, "box-drawing symbol ◆ = width 1"},
	}

	for _, tt := range tests {
		t.Run(tt.desc, func(t *testing.T) {
			got := IsThaiZeroWidth(tt.rune)
			if got != tt.isZero {
				t.Errorf("IsThaiZeroWidth(%q U+%04X) = %v, want %v — %s",
					string(tt.rune), tt.rune, got, tt.isZero, tt.desc)
			}
		})
	}
}

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
