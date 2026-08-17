//go:build !windows

package terminal

// SetUTF8Console is a no-op on non-Windows platforms.
// Linux and macOS terminals use UTF-8 by default — no codepage switch needed.
func SetUTF8Console() {
	// Nothing to do — UTF-8 is the default on Unix-like systems
}
