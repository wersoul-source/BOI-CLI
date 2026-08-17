# 🔄 BOI CLI — Product Lifecycle

> ออกแบบ: 31 กรกฎาคม 2026
> โดย: Kampun (คำปัน) — BOI Family

---

## Full Lifecycle

```
Research → Specification → Architecture → Implementation → Packaging
   → Installation → First Run → Upgrade → Daily Use → Troubleshooting
   → Uninstall
```

---

## Stage 1: Research ✅ COMPLETE

**What:** ศึกษา CLI tools ที่มีอยู่ + ดึง DNA มาใช้

**Deliverables:**
- ✅ 7 CLI tools analyzed (Hermes, OpenCode, Claude Code, Codex CLI, Antigravity, Agent Zero, ZeroClaw)
- ✅ Pattern extraction: ReAct loop, GEPA evolution, Memory hierarchy, PSC chain
- ✅ 13 installer tools studied (gh, claude, ollama, uv, bun, pnpm, homebrew, winget, scoop, cargo, go install, npm)
- ✅ Competitor gap analysis

**Key Decision:** Go binary + Chimera Architecture (ไม่ fork source code, เอาแต่ pattern)

---

## Stage 2: Specification ✅ COMPLETE

**What:** กำหนด scope, features, persona system, evolution model

**Deliverables:**
- ✅ PLAN.md — แผนแม่บทสมบูรณ์
- ✅ 8-Layer Architecture defined
- ✅ Greek God Level System (9 levels × 3 tiers)
- ✅ 10 Evidence Dimensions
- ✅ Command specification (7 top-level + subcommands)
- ✅ Persona system design (6 profiles)
- ✅ PSC (Provider Supply Chain) specification
- ✅ Phantom DB memory design

---

## Stage 3: Architecture ✅ COMPLETE

**What:** Tech stack, project structure, layer contracts

**Deliverables:**
- ✅ Tech stack: Go 1.24+ | Cobra | Bubbletea | SQLite | YAML
- ✅ Project structure: cmd/ + internal/ pattern
- ✅ Internal package boundaries: cli, config, llm, persona, skill, memory, agent, tui, weight
- ✅ Module contract: `go.mod` with minimal dependencies (5 direct)
- ✅ Build system: Makefile + go build

**Key Decision:** Pure Go runtime, Python for AI/vision/research only

---

## Stage 4: Implementation ✅ COMPLETE (v0.1.0)

**What:** Build the actual code

**Phases delivered:**

| Phase | System | Status |
|-------|--------|--------|
| Phase 1 | Core Runtime (CLI, Config, Workspace, Logger, Command) | ✅ |
| Phase 2 | PSC Mini (Provider interface, 4 providers, auto-fallback) | ✅ |
| Phase 3 | Persona System (6 personas, registry, loader, CLI) | ✅ |
| Phase 4 | Skill System (runtime, loader, builtins, MCP client) | ✅ |
| Phase 5 | Phantom DB Memory (SQLite, FTS5, context, repomap) | ✅ |
| Phase 6 | Agent Loop (ReAct: plan, execute, review, learn) | ✅ |
| TUI | Bubbletea terminal interface (splash, chat, 8-bit art) | ✅ |

**Binary:** ~8 MB (Go, Windows/Linux/macOS, amd64)

---

## Stage 5: Packaging ⬜ IN PROGRESS

**What:** Build binaries, generate checksums, create install scripts

**Deliverables needed:**

| Artifact | Status | File |
|----------|--------|------|
| Windows binary | ✅ | `bin/boi.exe` |
| macOS amd64 binary | ⬜ | `bin/boi-darwin-amd64` |
| macOS arm64 binary | ⬜ | `bin/boi-darwin-arm64` |
| Linux amd64 binary | ⬜ | `bin/boi-linux-amd64` |
| Linux arm64 binary | ⬜ | `bin/boi-linux-arm64` |
| SHA256SUMS.txt | ⬜ | Generated per release |
| Windows installer script | ✅ | `scripts/install.ps1` |
| Unix installer script | ✅ | `scripts/install.sh` |
| Homebrew formula | ⬜ | `Formula/boi.rb` → homebrew-boi-cli repo |
| WinGet manifest | ⬜ | `BOIFamily.BOICLI.yaml` → winget-pkgs |
| Scoop manifest | ⬜ | `boi.json` → scoop-bucket |
| Docker image | ⬜ | `ghcr.io/boi-family/boi-cli` |
| Release notes | ⬜ | CHANGELOG.md |

**Build pipeline (GitHub Actions):**

```yaml
name: Release
on:
  push:
    tags: ['v*']
jobs:
  build:
    strategy:
      matrix:
        os: [ubuntu-latest, macos-latest, windows-latest]
        arch: [amd64, arm64]
    steps:
      - uses: actions/checkout@v4
      - uses: actions/setup-go@v5
        with: { go-version: '1.24' }
      - run: go build -o bin/boi ./cmd/boi
        env:
          GOOS: ${{ matrix.os }}
          GOARCH: ${{ matrix.arch }}
      - uses: actions/upload-artifact@v4
  checksum:
    needs: build
    runs-on: ubuntu-latest
    steps:
      - run: sha256sum bin/* > SHA256SUMS.txt
      - uses: actions/upload-release-asset
  publish:
    needs: checksum
    # Publish to:
    # - GitHub Releases (primary)
    # - Homebrew formula update
    # - WinGet manifest update
    # - Docker push
```

---

## Stage 6: Installation ⬜ TO BUILD

**What:** User gets BOI CLI onto their machine

**Methods (see INSTALLER_FLOW.md for full flow):**

1. **One-liner script** (primary)
   ```
   Windows:  irm https://boi.sh/install.ps1 | iex
   macOS:    curl -fsSL https://boi.sh/install.sh | bash
   Linux:    curl -fsSL https://boi.sh/install.sh | bash
   ```

2. **Package managers** (secondary)
   ```
   brew install boi-family/boi-cli
   winget install BOIFamily.BOICLI
   scoop install boi
   go install github.com/boi-family/boi-cli/cmd/boi@latest
   ```

3. **Manual** (tertiary)
   ```
   Download from GitHub Releases → extract → add to PATH
   ```

**Prerequisites:** None (Go not required for prebuilt binary)

---

## Stage 7: First Run ✅ DESIGNED

**What:** User's first interaction after install

**Flow (see FIRST_RUN.md for full flow):**
1. Auto-detect first run (no `.boi/`)
2. Auto-init workspace (silent)
3. Splash screen (BOI ASCII art)
4. Prompt for `.env` setup (optional, skippable)
5. `boi doctor` health check
6. TUI launches

**Key principle:** Zero friction — `boi` just works immediately, no config required.

---

## Stage 8: Upgrade ⬜ TO BUILD

**What:** User updates to newer version

**Mechanism:** `boi upgrade`

1. Check current version (`boi version`)
2. Fetch latest from GitHub Releases API
3. Download + verify SHA256
4. Backup current binary (`boi.old`)
5. Replace binary atomically
6. Verify new version

**Also:** Homebrew `brew upgrade`, WinGet `winget upgrade`, Go `go install@latest`

---

## Stage 9: Daily Use ✅ CURRENT

**What:** Developer uses BOI CLI in daily workflow

**Typical session:**
```bash
cd my-project
boi                            # Launch TUI
boi ask "explain this code"    # Quick question
boi ask -p dang "debug auth"   # Debug with specialist
boi run "git diff"             # Execute commands
boi memory search "deploy"     # Find past knowledge
```

**Background operations:**
- Memory compaction (automatic, during idle)
- Weight recalculation (after each session)
- Repomap refresh (on directory change detection)

---

## Stage 10: Troubleshooting ⬜ TO BUILD

**What:** User encounters issues

**Tools:**

| Tool | Purpose | Status |
|------|---------|--------|
| `boi doctor` | Health check (binary, config, PSC, memory) | ⬜ v0.2.0 |
| `boi doctor --fix` | Auto-fix common issues | ⬜ v0.3.0 |
| `boi --verbose` | Verbose mode for debugging | ✅ |
| `boi config --all` | View full configuration | ✅ |
| Error codes | Standardized exit codes | ✅ (0–4) |
| Logs | Structured logging (slog) | ✅ |
| FAQs | Common issues in docs | ⬜ |

**Common issues + solutions:**

| Issue | Cause | Fix |
|-------|-------|-----|
| `boi not found` | ~/.boi/bin not in PATH | Re-run install script / add manually |
| `no config found` | .boi/ not initialized | `boi init` |
| `provider error` | Wrong API key or rate limit | `boi config --all` → check keys |
| `memory corruption` | SQLite DB damaged | `rm .boi/memory/*.db && boi init --force` |
| Permission denied | Binary not executable | `chmod +x ~/.boi/bin/boi` |

---

## Stage 11: Uninstall ⬜ TO DOCUMENT

**What:** User removes BOI CLI

```
# Windows
Remove-Item -Recurse -Force $env:USERPROFILE\.boi

# macOS / Linux
rm -rf ~/.boi
```

**Also remove PATH entry:**
- Windows: System → Environment Variables → Path → remove `.boi\bin`
- macOS/Linux: remove `export PATH="$HOME/.boi/bin:$PATH"` from `~/.bashrc`/`~/.zshrc`

**Package manager uninstall:**
```bash
brew uninstall boi-cli
winget uninstall BOIFamily.BOICLI
scoop uninstall boi
```

**What gets left behind (intentional):**
- Nothing. `~/.boi` contains everything. No registry entries, no system files, no LaunchAgents.

---

## Lifecycle Summary Table

| Stage | Status | Key Artifacts | Done? |
|-------|--------|---------------|-------|
| 1. Research | Complete | INSTALL_PLAN.md, DNA analysis | ✅ |
| 2. Specification | Complete | PLAN.md | ✅ |
| 3. Architecture | Complete | go.mod, project structure | ✅ |
| 4. Implementation | Complete | v0.1.0 binary (8 MB) | ✅ |
| 5. Packaging | In Progress | CI/CD, binaries for all platforms | ⬜ |
| 6. Installation | Designed | install.ps1, install.sh, formulas | ⬜ |
| 7. First Run | Designed | SPLASH, auto-init, doctor | ✅ code exists |
| 8. Upgrade | Planned | `boi upgrade` command | ⬜ v0.2.0 |
| 9. Daily Use | Active | All commands working | ✅ |
| 10. Troubleshooting | Planned | `boi doctor` | ⬜ v0.2.0 |
| 11. Uninstall | Designed | `rm -rf ~/.boi` | ✅ simple |

---

## Version Roadmap

```
v0.1.0  ████████████████████  Core Runtime (current)
v0.2.0  ░░░░░░░░░░░░░░░░░░░░  Installer + boi doctor + boi upgrade
v0.3.0  ░░░░░░░░░░░░░░░░░░░░  Multi-Agent + GEPA Evolution
v0.4.0  ░░░░░░░░░░░░░░░░░░░░  Plugin System + Community Skills
v0.5.0  ░░░░░░░░░░░░░░░░░░░░  IDE Integration + VS Code Extension
v1.0.0  ░░░░░░░░░░░░░░░░░░░░  Stable Release
```

---

*สิ้นสุด PRODUCT_LIFECYCLE.md*
