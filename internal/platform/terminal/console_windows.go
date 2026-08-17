//go:build windows

package terminal

import (
	"golang.org/x/sys/windows"
)

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
	const cpUTF8 = 65001
	windows.SetConsoleOutputCP(cpUTF8)
	windows.SetConsoleCP(cpUTF8)
}
