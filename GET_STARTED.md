# Get Started with BOI CLI

See the [README](README.md) for the full guide.

**TL;DR** — 3 steps:

### Install

**Linux/macOS:**
```bash
curl -fsSL https://raw.githubusercontent.com/wersoul-source/BOI-CLI/main/scripts/install.sh | bash
```

**Windows:**
```powershell
irm https://raw.githubusercontent.com/wersoul-source/BOI-CLI/main/scripts/install.ps1 | iex
```

**Or download from [GitHub Releases](https://github.com/wersoul-source/BOI-CLI/releases/latest):**
- `boi_0.3.0_linux_amd64.tar.gz` — Ubuntu/Debian x86_64
- `boi_0.3.0_linux_arm64.tar.gz` — Raspberry Pi, ARM servers
- `boi_0.3.0_darwin_amd64.tar.gz` — macOS Intel
- `boi_0.3.0_darwin_arm64.tar.gz` — macOS Apple Silicon
- `boi_0.3.0_windows_amd64.tar.gz` — Windows x64
- `boi_0.3.0_windows_arm64.tar.gz` — Windows ARM

### Setup
```bash
boi setup    # Configure providers (interactive TUI wizard)
```

### Launch
```bash
boi          # Launch TUI
boi ask "hello"  # Or CLI mode
```

That's it. BOI CLI handles the rest — persona selection, chat, command palette, memory, and provider auto-fallback.

For detailed documentation, see:
- `docs/` — CLI commands, architecture, system flow
- `/help` in TUI — keyboard shortcuts and commands
