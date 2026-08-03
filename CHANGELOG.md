# Changelog

All notable changes to BOI CLI.

## v0.3.0 (2026-08-03) — Cross-platform Release + Full Audit

### Cross-platform Support
- Build for **Windows** (amd64, arm64), **Linux** (amd64, arm64), **macOS** (amd64, arm64)
- Refactored `internal/term/` with Go build tags — no more Windows-only imports
- `SetUTF8Console()` now a no-op on Unix (UTF-8 is default)
- `EnsureThaiFont()` now a no-op on Unix (system fonts available)

### Bug Fixes
- **Wizard keyboard**: Fixed — textInput.Update() was never called in stepAPIKey
- **Doctor false-positive**: Fixed — "All checks passed!" printed even on fail
- **Doctor count mismatch**: Fixed — counted 4 providers, factory supports 20
- **Font error handling**: Fixed — AddFontResourceW return now checked
- **Version override**: Fixed — changed `Version` from `const` to `var` so ldflags works

### Infrastructure
- Release v0.3.0 includes binary archives for all 6 platforms + checksums
- `install.sh` (Linux/macOS) auto-fetches latest release from GitHub
- `install.ps1` (Windows) auto-fetches latest release (no more hardcoded version)
- Full audit of 70+ source files — 4 bugs found and fixed

## v0.2.0 (2026-08-01) — Thai Terminal Fix + PSC Rotation
- Thai terminal rendering fix (codepage 874 → UTF-8, ThaiStringWidth, font install)
- PSC (Provider Supply Chain) auto-rotation with usage bar
- TUI improvements: responsive input, Thai-safe chat bubbles

## v0.1.2 (2026-07-31) — Provider Test + Setup Wizard
- Provider test after setup — validates API keys immediately
- Complete onboarding wizard

## v0.1.1 (2026-07-31)
- Bug fixes and improvements

## v0.1.0 (2026-07-31) — Initial Release
- Core CLI with TUI, 6 personas, Phantom DB, PSC rotation
