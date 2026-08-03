//go:build windows

package term

import (
	"fmt"
	"io"
	"net/http"
	"os"
	"path/filepath"
	"strings"
	"time"
	"unsafe"

	"golang.org/x/sys/windows"
	"golang.org/x/sys/windows/registry"
)

const (
	// Noto Sans Thai variable font — OFL license, verified 200 OK
	notoSansThaiURL = "https://raw.githubusercontent.com/google/fonts/main/ofl/notosansthai/NotoSansThai%5Bwdth%2Cwght%5D.ttf"
)

// ThaiFontInstalled reports whether at least one Thai-capable font is
// registered in the system. On Windows 8+ Leelawadee UI is present —
// this is the fast path (no network needed).
func ThaiFontInstalled() bool {
	k, err := registry.OpenKey(registry.LOCAL_MACHINE,
		`SOFTWARE\Microsoft\Windows NT\CurrentVersion\Fonts`, registry.QUERY_VALUE)
	if err != nil {
		return false
	}
	defer k.Close()

	vals, err := k.ReadValueNames(-1)
	if err != nil {
		return false
	}

	for _, v := range vals {
		switch {
		case v == "Leelawadee UI (TrueType)",
			v == "Leelawadee UI Bold (TrueType)",
			v == "Leelawadee (TrueType)",
			v == "Noto Sans Thai (TrueType)",
			v == "Tahoma (TrueType)", // ships with Thai on modern Windows
			strings.HasPrefix(v, "Angsana"),
			strings.HasPrefix(v, "Kodchiang"):
			return true
		}
	}
	return false
}

// EnsureThaiFont installs Noto Sans Thai if no Thai-capable font is found.
// Returns (installed=true, nil) if the font was just installed,
// or (false, nil) if a Thai font was already present.
//
// The install is user-level (no admin required):
//   - Downloads the variable font (~500KB)
//   - Writes to %LOCALAPPDATA%\Microsoft\Windows\Fonts
//   - Registers in HKCU\...\Fonts registry
//   - Calls AddFontResourceW + broadcasts WM_FONTCHANGE
func EnsureThaiFont() (installed bool, err error) {
	if ThaiFontInstalled() {
		return false, nil
	}

	dir := os.Getenv("LOCALAPPDATA")
	if dir == "" {
		return false, fmt.Errorf("LOCALAPPDATA not set")
	}
	fontDir := filepath.Join(dir, `Microsoft\Windows\Fonts`)
	if err := os.MkdirAll(fontDir, 0o755); err != nil {
		return false, fmt.Errorf("create font dir: %w", err)
	}

	dst := filepath.Join(fontDir, "NotoSansThai.ttf")
	if err := downloadFile(notoSansThaiURL, dst); err != nil {
		return false, fmt.Errorf("download Noto Sans Thai: %w", err)
	}

	if err := registerFont(dst); err != nil {
		return false, fmt.Errorf("register font: %w", err)
	}

	return true, nil
}

func downloadFile(url, dst string) error {
	client := &http.Client{Timeout: 30 * time.Second}
	resp, err := client.Get(url)
	if err != nil {
		return err
	}
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusOK {
		return fmt.Errorf("HTTP %d from %s", resp.StatusCode, url)
	}

	out, err := os.Create(dst)
	if err != nil {
		return err
	}
	defer out.Close()

	if _, err := io.Copy(out, resp.Body); err != nil {
		os.Remove(dst)
		return err
	}
	return nil
}

func registerFont(path string) error {
	// 1. Write HKCU registry (persistent per-user install)
	k, _, err := registry.CreateKey(registry.CURRENT_USER,
		`SOFTWARE\Microsoft\Windows NT\CurrentVersion\Fonts`, registry.SET_VALUE)
	if err != nil {
		return err
	}
	defer k.Close()

	if err := k.SetStringValue("Noto Sans Thai (TrueType)", path); err != nil {
		return err
	}

	// 2. Load into current session
	pathPtr, _ := windows.UTF16PtrFromString(path)
	ret, _, err := procAddFontResourceW.Call(uintptr(unsafe.Pointer(pathPtr)))
	if ret == 0 {
		return fmt.Errorf("AddFontResourceW failed: %w", err)
	}

	// 3. Notify all windows (so apps pick up the font without restart)
	const (
		hwndBroadcast = 0xFFFF
		wmFontChange  = 0x001D
	)
	procSendMessageW.Call(hwndBroadcast, wmFontChange, 0, 0)
	// SendMessage error is non-critical — font still loaded even if broadcast fails

	return nil
}

var (
	user32               = windows.NewLazySystemDLL("user32.dll")
	gdi32                = windows.NewLazySystemDLL("gdi32.dll")
	procAddFontResourceW = gdi32.NewProc("AddFontResourceW")
	procSendMessageW     = user32.NewProc("SendMessageW")
)
