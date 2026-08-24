# S25+ Mobile Runbook

Target: Android ARM64 using Termux or an equivalent terminal environment. Physical-device acceptance is required.

## Preferred path: build on device

```sh
pkg update
pkg install git golang
go version
git clone https://github.com/wersoul-source/BOI-CLI.git
cd BOI-CLI
git rev-parse HEAD
go test ./...
go build -trimpath -o "$PREFIX/bin/boi" ./cmd/boi
boi version
boi doctor
```

The installed Go toolchain must satisfy the `go.mod` requirement. Do not downgrade the module declaration only to force a build.

## Workspace smoke test

```sh
mkdir -p "$HOME/boi-mobile-test"
cd "$HOME/boi-mobile-test"
boi init
boi doctor
boi
```

After Provider configuration, test:

```sh
boi ask "ตอบคำว่า BOI-MOBILE-READY เท่านั้น" --verbose
```

Do not paste credentials into shell history, logs, screenshots, commits, or handoff documents.

## Acceptance checklist

- Exact Git commit recorded.
- `boi version` starts.
- `boi doctor` identifies the Android environment without panic.
- TUI renders and exits cleanly.
- Thai input and output render correctly.
- Ctrl+C/Esc cancellation behaves as documented by the terminal.
- Workspace read remains inside the selected root.
- Offline startup reports degraded capability without downloading.
- Provider request succeeds when network is enabled.
- `boi ask` behavior and exit status are recorded.

## Current evidence and limits

Windows cross-build for `android/arm64` passed. This proves compilation only. It does not prove Termux execution, Android storage permission behavior, keyboard events, TUI dimensions, DNS/TLS, battery behavior, or Automation readiness.

## Automation boundary

Use read-only experiments first. Current non-interactive Agent Service does not approve write, process, or external calls. Stable JSON output and final exit-code contracts are still planned, so parsers built before Phase H6 are provisional.
