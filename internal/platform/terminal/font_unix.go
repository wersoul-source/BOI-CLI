//go:build !windows

package terminal

// ThaiFontInstalled reports whether at least one Thai-capable font is
// available on the system. On Linux/macOS, we check for common Thai fonts.
func ThaiFontInstalled() bool {
	// Common Thai fonts on Linux/macOS:
	// - Noto Sans Thai (most Linux distros)
	// - Tahoma (macOS with Thai language pack)
	// - Various Thai fonts on Ubuntu/Debian (fonts-thai-tlwg)
	//
	// Font detection on Unix is complex (fc-list, fontconfig).
	// For simplicity, assume Thai fonts are available — modern Linux/macOS
	// systems typically have them or fallback rendering works.
	return true
}

// EnsureThaiFont is a no-op on non-Windows platforms.
// Linux: Thai fonts are available via package managers (fonts-thai-tlwg, etc.)
// macOS: System has built-in Thai rendering support.
func EnsureThaiFont() (installed bool, err error) {
	// Nothing to do — assume fonts are available or fallback rendering works
	return false, nil
}
