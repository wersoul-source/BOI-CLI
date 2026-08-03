// Package term handles terminal-level fixes for Thai (and other non-Latin)
// text display. It covers 3 layers: encoding (codepage), font detection,
// and width calculation (see thai-terminal-fix skill for full theory).
//
// On Windows, the console defaults to codepage 874 (TIS-620), but Go writes
// UTF-8 bytes. This mismatch causes Thai text to render as garbled characters.
// SetUTF8Console fixes this at the process level — no system-wide changes needed.
//
// Platform-specific code is split into separate files:
//
//	console_windows.go  — Windows (SetConsoleOutputCP, registry)
//	console_unix.go     — Linux/macOS (no-op, UTF-8 is default)
//	font_windows.go     — Windows font installation
//	font_unix.go        — Linux/macOS (no-op or native method)
package term

import (
	"regexp"
	"strings"
	"unicode"
)

var ansiRegex = regexp.MustCompile(`\x1b\[[0-9;]*[a-zA-Z]`)

// StripANSI removes ANSI escape sequences from s.
func StripANSI(s string) string {
	return ansiRegex.ReplaceAllString(s, "")
}

// ThaiZeroWidthRunes lists Thai Unicode characters that are zero-width
// combining marks (vowels above/below, tone marks). go-runewidth counts
// these as width=1 (incorrect), but terminals render them as width=0
// (overlapping the previous consonant). This is a known bug in go-runewidth
// v0.0.19 and earlier — see thai-terminal-fix skill, EXP-A.
var ThaiZeroWidthRunes = map[rune]bool{
	0x0E31: true, // MAI HAN-AKAT
	0x0E34: true, // SARA I
	0x0E35: true, // SARA II
	0x0E36: true, // SARA UE
	0x0E37: true, // SARA UEE
	0x0E38: true, // SARA U
	0x0E39: true, // SARA UU
	0x0E3A: true, // PHINTHU
	0x0E47: true, // MAITAIKHU
	0x0E48: true, // MAI EK
	0x0E49: true, // MAI THO
	0x0E4A: true, // MAI TRI
	0x0E4B: true, // MAI CHATTAWA
	0x0E4C: true, // THANTHAKHAT
	0x0E4D: true, // NIKHAHIT
	0x0E4E: true, // YAMAKKAN
}

// IsThaiZeroWidth reports whether r is a Thai zero-width combining mark.
func IsThaiZeroWidth(r rune) bool {
	return ThaiZeroWidthRunes[r]
}

// ThaiStringWidth returns the display width of s, correcting for the
// go-runewidth bug that counts Thai marks as width=1.
// Use this instead of runewidth.StringWidth for Thai text.
// Automatically strips ANSI escape sequences.
func ThaiStringWidth(s string) int {
	s = StripANSI(s)
	width := 0
	for _, r := range s {
		if IsThaiZeroWidth(r) {
			continue // skip Thai zero-width marks
		}
		// For other runes, use the standard width calculation
		if isWideRune(r) {
			width += 2
		} else {
			width += 1
		}
	}
	return width
}

// isWideRune reports whether r should be displayed as 2 columns (CJK, etc.).
func isWideRune(r rune) bool {
	return unicode.Is(unicode.Han, r) ||
		unicode.Is(unicode.Hangul, r) ||
		r >= 0xFF01 && r <= 0xFF60 || // Fullwidth ASCII
		r >= 0xFFE0 && r <= 0xFFE6 // Fullwidth symbols
}

// NormalizeThai converts precomposed Thai vowel+tone sequences to proper
// logical order for display. This handles cases where tone marks appear
// before vowels in the byte stream (rare, but can happen with copy-paste).
func NormalizeThai(s string) string {
	// Most Thai text is already in correct order; this handles edge cases
	// where combining marks may be reordered.
	// For now, return as-is — can be enhanced later.
	return s
}

// IsThai returns true if s contains any Thai Unicode characters.
func IsThai(s string) bool {
	for _, r := range s {
		if unicode.Is(unicode.Thai, r) {
			return true
		}
	}
	return false
}

// ThaiConsonantCount counts only Thai consonants in s (ignoring vowels,
// tone marks, and other combining characters).
func ThaiConsonantCount(s string) int {
	count := 0
	for _, r := range s {
		if r >= 0x0E01 && r <= 0x0E2E && !IsThaiZeroWidth(r) {
			count++
		}
	}
	return count
}

// TruncateThai truncates s to maxWidth display columns, respecting Thai
// zero-width combining marks (they don't count toward width).
func TruncateThai(s string, maxWidth int) string {
	width := 0
	for i, r := range s {
		w := 1
		if isWideRune(r) {
			w = 2
		}
		if IsThaiZeroWidth(r) {
			w = 0
		}
		if width+w > maxWidth {
			return s[:i]
		}
		width += w
	}
	return s
}

// PadRight pads s with spaces to reach targetWidth display columns.
// Uses ThaiStringWidth for accurate Thai text measurement.
func PadRight(s string, targetWidth int) string {
	w := ThaiStringWidth(s)
	if w >= targetWidth {
		return s
	}
	return s + strings.Repeat(" ", targetWidth-w)
}
