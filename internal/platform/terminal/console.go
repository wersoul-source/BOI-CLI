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
package terminal

import "regexp"

var ansiRegex = regexp.MustCompile(`\x1b\[[0-9;]*[a-zA-Z]`)

// StripANSI removes ANSI escape sequences from s.
func StripANSI(s string) string {
	return ansiRegex.ReplaceAllString(s, "")
}
