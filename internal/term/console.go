// Package term handles terminal-level fixes for Thai (and other non-Latin)
// text display. It covers 3 layers: encoding (codepage), font detection,
// and width calculation (see thai-terminal-fix skill for full theory).
//
// On Windows, the console defaults to codepage 874 (TIS-620), but Go writes
// UTF-8 bytes. This mismatch causes Thai text to render as garbled characters.
// SetUTF8Console fixes this at the process level — no system-wide changes needed.
package term

import (
	"regexp"
	"unicode"

	"golang.org/x/sys/windows"
)

var ansiRegex = regexp.MustCompile(`\x1b\[[0-9;]*[a-zA-Z]`)

// StripANSI removes ANSI escape sequences from s.
func StripANSI(s string) string {
	return ansiRegex.ReplaceAllString(s, "")
}

// SetUTF8Console switches the process console to UTF-8 (codepage 65001).
//
// Go's os.Stdout writes UTF-8 bytes. If the console is on codepage 874
// (Thai Windows default), those bytes are interpreted as TIS-620 → garbled.
// This fix:
//   - Works in every terminal: WT, cmd, mintty, wezterm, VS Code, etc.
//   - Affects only this process — no registry or system changes
//   - Safe to call multiple times
//   - Must be called early in main(), before any fmt.Println/Print
func SetUTF8Console() {
	// CP_UTF8 = 65001, defined as a constant since x/sys/windows
	// may not export it in all versions
	const cpUTF8 = 65001
	windows.SetConsoleOutputCP(cpUTF8)
	windows.SetConsoleCP(cpUTF8)
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
		// For other runes, use the standard runewidth
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
	// CJK Unified Ideographs, Fullwidth Forms, etc.
	// Simplified check — go-runewidth handles this better
	// but we don't need it for Thai focus
	return unicode.Is(unicode.Han, r) ||
		unicode.Is(unicode.Hangul, r) ||
		r >= 0xFF01 && r <= 0xFF60 || // Fullwidth ASCII
		r >= 0xFFE0 && r <= 0xFFE6 // Fullwidth symbols
}
