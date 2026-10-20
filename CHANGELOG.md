# Changelog

All notable changes to BOI CLI.

## v1.6.0 — Adaptive Execution (unreleased)

### Native tool calling
- Providers can carry native tool definitions (`llm.ToolCaller`); the OpenAI-compatible adapter sends `tools` and reads `tool_calls`
- Native mode is used only when every Router candidate supports it and the `native_tool_schema` probe proves the model calls tools correctly; otherwise the `<boi-action>` envelope remains, and an envelope is still accepted in native mode
- Provider call IDs are replaced with host IDs, so Providers that reuse `call_0` cannot collide with idempotency keys

### Batch execution
- One turn may propose up to 5 Tool calls
- A batch made only of automatic reads runs concurrently; any change, process or external call keeps the batch sequential, and each call is still authorized, executed and verified on its own
- A denied or failed call stops the rest of the batch

### Context lifecycle (compute once, reference many)
- Every tool output is kept in a per-task context store with a checksum
- An output above 16 KB keeps its head and tail and names the reference and line range of the omitted middle; for `workspace.read` the file content is shortened inside the JSON, so line numbers refer to the file and fields such as `Truncated` stay intact
- When the 48 KB observation budget is spent, older observations become stubs with their reference instead of disappearing
- New Runtime Tool `context.expand(ref, lines)` reads any stored output back exactly, after an integrity check; it is added to every non-empty active set that has room
- No lossy compression: nothing is summarized or abbreviated, so an answer built from expanded context sees the original text

### Planning
- The static Task Plan is no longer sent to the model every turn; it stays in the task manifest

### Qualification
- Probes judge behavior instead of exact text: prose around a correct answer passes, a forbidden outcome fails, an empty reply is `inconclusive` and enables nothing
- `native_tool_schema` is now measured: `unsupported` (adapter or endpoint), `failed` (model ignores tools) or `passed`
- Probe suite is `boi-provider-v2`: **run `boi provider qualify <name>` again after upgrading**

### Measurement
- `scripts/acceptance/benchmark.sh` runs a fixed read-only task set against a real Provider and can compare two versions (`BASE_REF=...`)

## v1.5.0 — BOI Agent Suite v1.5

### Architecture
- Fixed spine (`core`, `runtime`) and pluggable Blocks (`equipment`, `service`, `agentfolder`, `subagent`)
- `block/port.Tool` contract; the Broker dispatches through a registered tool table
- Built-in workspace, process and MCP Tools moved to Equipment and verify their own effects
- `app.BuildAgent` is the single composition point for TUI and CLI
- Dependency rules enforced by `internal/architecture/deps_test.go`

### Fixes
- Provider adapters: request timeout, 8 MB response cap; Google API key moved from URL to header
- Memory store rejects path-traversal IDs; persona names match case-insensitively
- Upgrade caps extracted archive entries at 512 MB
- Command deny-list hardened and documented as best-effort, not isolation
- `doctor` reports Config and Agent identity as separate checks

### Build
- Linux is the release gate: race tests, staticcheck, coverage floor, built-binary folder simulation
- Other platforms are compile checks; `make smoke` runs the Linux acceptance on a VM

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
