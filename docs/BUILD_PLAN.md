# BOI CLI -- Build Plan: Installer & Missing Features

> Version: v0.1.0
> Updated: 31 July 2026
> By: Kampun (Kampun) -- BOI Family

---

## Overview

This document tracks everything needed to complete the BOI CLI installer ecosystem and fill feature gaps identified during v0.1.0 development. Each task includes: what to build, why it matters, effort estimate, and dependencies.

---

## Implementation Tasks

| # | Task | Priority | Status | Effort | Depends On |
|---|------|----------|--------|--------|------------|
| I-1 | `scripts/install.ps1` (Windows installer) | RED High | Template done | 2h | I-6, I-7 |
| I-2 | `scripts/install.sh` (macOS/Linux) | RED High | Template done | 2h | I-6, I-7 |
| I-3 | `boi upgrade` command (self-update) | RED High | Not started | 4h | I-6 |
| I-4 | `boi doctor` command (health check) | YELLOW Medium | Not started | 3h | None |
| I-5 | `boi uninstall` command | YELLOW Medium | Not started | 2h | I-6 |
| I-6 | GitHub Releases + goreleaser | RED High | Not started | 3h | I-14 |
| I-7 | Checksum generation (SHA256) | RED High | Not started | 1h | I-14 |
| I-8 | Homebrew formula | YELLOW Medium | Not started | 2h | I-6, I-7 |
| I-9 | WinGet manifest | YELLOW Medium | Not started | 2h | I-6, I-7 |
| I-10 | Scoop bucket | GREEN Low | Not started | 1h | I-6, I-7 |
| I-11 | `boi version` subcommand fix | YELLOW Medium | Bug | 1h | None |
| I-12 | `.env` auto-detect + prompt | YELLOW Medium | Not started | 3h | I-4 |
| I-13 | CI/CD pipeline (GitHub Actions) | RED High | Not started | 3h | I-14 |
| I-14 | Cross-platform build setup | RED High | Not started | 2h | None |

---

## Detailed Task Descriptions

---

### I-1: `scripts/install.ps1` (Windows Installer)

**What:** Production-ready PowerShell installer for Windows. Downloads the correct binary from GitHub Releases, verifies SHA256 checksum, installs to `~/.boi/bin/`, and adds to PATH.

**Why:** This is the primary install method for Windows users (Tier 1: one-liner). Must be bulletproof -- it's the first experience most Windows users will have with BOI CLI.

**Current state:** Template exists at `scripts/install.ps1` (272 lines) with full 8-step flow skeleton. Missing:
- `--silent` flag on `boi init` (init command needs updating)
- Tested on fresh Windows VM
- Error recovery for partial installs

**What to build:**
1. Finalize `install.ps1` script
2. Add `--silent` flag to `boi init` (`internal/cli/init.go`)
3. Test on clean Windows 10/11 VM
4. Test on Windows ARM64 (if available)
5. Document in README

**Effort:** 2 hours

**Dependencies:** I-6 (need GitHub Release URLs to be live), I-7 (need SHA256SUMS.txt)

**Files to modify:**
- `scripts/install.ps1` -- finalize
- `internal/cli/init.go` -- add `--silent` flag
- `README.md` -- add Windows install instructions

---

### I-2: `scripts/install.sh` (macOS/Linux Installer)

**What:** Production-ready bash installer for macOS and Linux. Detects OS/architecture, downloads binary, verifies checksum, installs, updates shell profile.

**Why:** Primary install method for macOS/Linux users. Must handle bash/zsh/fish shell detection and respect XDG conventions.

**Current state:** Template exists at `scripts/install.sh` (327 lines) with full flow. Missing:
- `--silent` flag on `boi init`
- Tested on macOS (arm64 + amd64)
- Tested on Linux (Ubuntu, Debian, Fedora, Arch)
- fish shell PATH injection format

**What to build:**
1. Finalize `install.sh` script
2. Test on macOS 14+ (arm64)
3. Test on Ubuntu 24.04 (amd64)
4. Test fish shell detection and config update
5. Document in README

**Effort:** 2 hours

**Dependencies:** I-6, I-7, I-1 (shared `--silent` init flag)

**Files to modify:**
- `scripts/install.sh` -- finalize
- `README.md` -- add macOS/Linux install instructions

---

### I-3: `boi upgrade` Command (Self-Update)

**What:** Built-in command to check for newer releases and self-update the binary atomically.

**Usage:**
```bash
boi upgrade             # Update to latest
boi upgrade v0.2.0      # Pin to specific version
boi upgrade --check     # Dry run: check only
```

**Why:** Critical for user retention. Without self-update, users must manually re-download from GitHub Releases. Every mature CLI (uv, bun, gh) has this.

**What to build:** Full implementation in `internal/cli/upgrade.go`:

```
boi upgrade flow:
  1. Read current version (embedded via ldflags at build)
  2. GET https://api.github.com/repos/wersoul-source/BOI-CLI/releases/latest
  3. Compare version strings (semver)
  4. If same: "Already up to date"
  5. If newer: prompt [Y/n]
  6. Download new binary to ~/.boi/bin/boi-new
  7. Backup current: mv boi -> boi.old
  8. Verify SHA256 from release
  9. Atomic replace: mv boi-new -> boi
  10. Verify: run `boi --version` to confirm
  11. Success message + keep boi.old as fallback
```

**Edge cases:**
- Binary is currently running (Windows: can replace, Linux: can't -- use temp + move)
- No internet connection (graceful error)
- Checksum mismatch (abort, keep old version)
- Version format: must parse semver `v0.1.0` vs `0.1.0`

**Effort:** 4 hours

**Dependencies:** I-6 (need releases published), I-14 (version must be embedded via `-ldflags`)

**Files to create/modify:**
- `internal/cli/upgrade.go` -- new file
- `internal/cli/root.go` -- register upgrade command
- `cmd/boi/main.go` -- add `-ldflags` for version embedding
- `Makefile` -- add ldflags to build target

---

### I-4: `boi doctor` Command (Health Check)

**What:** System health check that verifies every component and reports status.

**Usage:**
```bash
boi doctor              # Run all checks
boi doctor --fix        # Auto-fix common issues
```

**Why:** First troubleshooting step for any user issue. Similar to `brew doctor`, `gh doctor`, or `docker info`. Reduces support burden by letting users self-diagnose.

**Checklist:**
```
[ ] BOI Binary
    [ ] Binary exists at expected path
    [ ] Binary is executable
    [ ] Version is readable (--version works)
    [ ] File size is reasonable (< 50 MB)

[ ] Go Installation (optional)
    [ ] `go version` works
    [ ] GOPATH is set
    [ ] GOPATH/bin is in PATH

[ ] Workspace (.boi/)
    [ ] .boi/ directory exists
    [ ] config.yaml is valid YAML
    [ ] config.yaml has required fields
    [ ] personas/ has at least 1 valid persona

[ ] PSC Providers
    [ ] .env file exists
    [ ] At least 1 PSC_1_NAME + PSC_1_API_KEY set
    [ ] Test connection to each provider (quick health ping)
    [ ] Report which providers are reachable

[ ] Memory (Phantom DB)
    [ ] .boi/memory/ directory exists
    [ ] Directory is writable
    [ ] JSON files are valid
    [ ] Memory count + breakdown (facts/patterns/solutions)

[ ] Skills
    [ ] .boi/skills/ directory exists
    [ ] Count of loaded skills
    [ ] Each skill file is valid markdown

[ ] Network
    [ ] Internet connectivity (can reach api.github.com)
    [ ] Can reach provider endpoints

[ ] PATH
    [ ] ~/.boi/bin is in PATH
    [ ] boi command resolves to correct binary
```

**Output format:**
```
BOI Doctor -- Health Check
===========================
  OK  Binary:     ~/.boi/bin/boi (v0.1.0, 6.8 MB)
  OK  Go:         1.24.2
  OK  Workspace:  .boi/ initialized
  OK  Config:     valid YAML
  OK  Personas:   6 loaded (boi, kamkaew, kampun, dang, don, kine)
  OK  Memory:     Phantom DB ready (5 entries)
  OK  Skills:     2 loaded (git, web)
  WARN Provider:  No API keys configured (simulated mode)
  OK  Network:    Connected
  OK  PATH:       boi resolves correctly

Overall: HEALTHY (no real AI -- set PSC_* in .env)
```

**Effort:** 3 hours

**Dependencies:** None

**Files to create/modify:**
- `internal/cli/doctor.go` -- new file
- `internal/cli/root.go` -- register doctor command

---

### I-5: `boi uninstall` Command

**What:** Clean removal of BOI CLI from the system.

**Usage:**
```bash
boi uninstall           # Remove everything
boi uninstall --dry-run # Show what would be removed
boi uninstall --keep-config # Remove binary, keep .boi/
```

**Why:** Users need a clean way to remove BOI CLI. Must handle: binary removal, PATH cleanup, .boi/ workspace removal, shell profile cleanup.

**What it removes:**
```
1. Binary:        ~/.boi/bin/boi (and boi.old)
2. PATH entry:    Remove from shell profile (.bashrc, .zshrc, .profile, config.fish)
                  Remove from Windows user PATH env var
3. Workspace:     ~/.boi/ (config, personas, skills, memory)
4. Optional:      .env file in project (prompt)
5. Optional:      .evolution/ data (if exists)
```

**Safety measures:**
- `--dry-run` shows everything before removing
- Confirmation prompt with list of paths
- Backup option: move to ~/.boi.bak instead of delete
- Detect if other tools share ~/.boi/ (unlikely but safe)

**Effort:** 2 hours

**Dependencies:** I-6 (for documentation linking to manual uninstall)

**Files to create/modify:**
- `internal/cli/uninstall.go` -- new file
- `internal/cli/root.go` -- register uninstall command

---

### I-6: GitHub Releases + GoReleaser

**What:** Automated release pipeline using GoReleaser. On every `v*` tag push, GoReleaser builds all platform binaries, generates checksums, creates GitHub Release, and uploads artifacts.

**Why:** Manual releases are error-prone and time-consuming. GoReleaser is the industry standard for Go CLI releases (used by gh, Hugo, Terraform, etc.).

**Configuration (`.goreleaser.yml`):**
```yaml
builds:
  - main: ./cmd/boi
    binary: boi
    goos: [windows, darwin, linux]
    goarch: [amd64, arm64]
    ldflags:
      - -s -w -X main.version={{.Version}} -X main.commit={{.Commit}} -X main.date={{.Date}}

archives:
  - format: tar.gz
    name_template: "{{ .ProjectName }}_{{ .Os }}_{{ .Arch }}"
    format_overrides:
      - goos: windows
        format: zip
    files:
      - README.md
      - LICENSE

checksum:
  name_template: 'SHA256SUMS.txt'
  algorithm: sha256

changelog:
  sort: asc

release:
  github:
    owner: wersoul-source
    name: BOI-CLI
  draft: false
  prerelease: auto
```

**What to build:**
1. Create `.goreleaser.yml` at project root
2. Test locally: `goreleaser release --snapshot --clean`
3. Create first GitHub Release manually to verify flow
4. Document release process in README

**Effort:** 3 hours

**Dependencies:** I-14 (cross-platform build must work), I-13 (CI/CD pipeline)

**Files to create/modify:**
- `.goreleaser.yml` -- new file
- `README.md` -- add release badge + download links

---

### I-7: Checksum Generation (SHA256)

**What:** SHA256 checksums for every release binary, published alongside releases.

**Why:** Security requirement for supply chain trust. Installer scripts verify checksums before installing. Package managers (Homebrew, WinGet, Scoop) require SHA256 hashes in their manifests.

**What to build:**
1. GoReleaser auto-generates `SHA256SUMS.txt` (see I-6)
2. Installer scripts parse and verify checksums
3. Document verification command for manual downloads

**Manual verification:**
```bash
# Windows
certutil -hashfile boi.exe SHA256

# macOS / Linux
shasum -a 256 boi
sha256sum boi
```

**Effort:** 1 hour (mostly GoReleaser config + documentation)

**Dependencies:** I-14 (binaries must exist), I-6 (GoReleaser config)

**Files to modify:**
- `.goreleaser.yml` -- checksum section
- `scripts/install.ps1` -- checksum verification (already templated)
- `scripts/install.sh` -- checksum verification (already templated)

---

### I-8: Homebrew Formula

**What:** Ruby formula file for installation via `brew install boi-family/boi-cli`.

**Why:** Primary package manager for macOS. Required for Tier 2 install strategy. Every major CLI tool has a Homebrew formula.

**Formula location:** Separate repo `boi-family/homebrew-boi-cli` with `Formula/boi.rb`.

```ruby
class Boi < Formula
  desc "BOI CLI -- AI Agent Runtime for the BOI Family"
  homepage "https://github.com/wersoul-source/BOI-CLI"
  version "0.1.0"
  license "MIT"

  on_macos do
    if Hardware::CPU.arm?
      url "https://github.com/wersoul-source/BOI-CLI/releases/download/v0.1.0/boi_Darwin_arm64.tar.gz"
      sha256 "PLACEHOLDER_ARM64"
    else
      url "https://github.com/wersoul-source/BOI-CLI/releases/download/v0.1.0/boi_Darwin_amd64.tar.gz"
      sha256 "PLACEHOLDER_AMD64"
    end
  end

  on_linux do
    if Hardware::CPU.arm?
      url "https://github.com/wersoul-source/BOI-CLI/releases/download/v0.1.0/boi_Linux_arm64.tar.gz"
      sha256 "PLACEHOLDER_LINUX_ARM64"
    else
      url "https://github.com/wersoul-source/BOI-CLI/releases/download/v0.1.0/boi_Linux_amd64.tar.gz"
      sha256 "PLACEHOLDER_LINUX_AMD64"
    end
  end

  def install
    bin.install "boi"
  end

  test do
    system "#{bin}/boi", "--version"
  end
end
```

**What to build:**
1. Create `boi-family/homebrew-boi-cli` GitHub repo
2. Create `Formula/boi.rb` with populated SHA256 hashes from first release
3. Test: `brew install boi-family/boi-cli`
4. Test: `brew test boi-cli`
5. Test: `brew audit --strict boi-cli`
6. Update formula on each release (or automate in CI)

**Effort:** 2 hours

**Dependencies:** I-6 (first release with published binaries + checksums), I-7 (SHA256 hashes)

**Files to create:**
- New repo: `boi-family/homebrew-boi-cli`
- `Formula/boi.rb`

---

### I-9: WinGet Manifest

**What:** YAML manifest for installation via `winget install BOIFamily.BOICLI`.

**Why:** Windows 10+ ships with WinGet pre-installed. Zero-friction install for Windows users. Part of Tier 2 strategy.

**Manifest location:** Submit PR to `microsoft/winget-pkgs` repository.

```yaml
PackageIdentifier: BOIFamily.BOICLI
PackageVersion: 0.1.0
PackageName: BOI CLI
Publisher: BOI Family
License: MIT
ShortDescription: AI Agent Runtime for the BOI Family
InstallerType: portable
Installers:
  - Architecture: x64
    InstallerUrl: https://github.com/wersoul-source/BOI-CLI/releases/download/v0.1.0/boi_Windows_amd64.zip
    InstallerSha256: PLACEHOLDER_SHA256
    Commands:
      - boi
  - Architecture: arm64
    InstallerUrl: https://github.com/wersoul-source/BOI-CLI/releases/download/v0.1.0/boi_Windows_arm64.zip
    InstallerSha256: PLACEHOLDER_SHA256
    Commands:
      - boi
ManifestType: singleton
ManifestVersion: 1.0.0
```

**What to build:**
1. Generate YAML manifest (can template from GoReleaser output)
2. Test locally: `winget install -m manifest.yaml`
3. Submit PR to microsoft/winget-pkgs
4. Document `winget install BOIFamily.BOICLI` in README
5. Update manifest on each release (or automate)

**Effort:** 2 hours (including PR review wait time)

**Dependencies:** I-6, I-7 (release + checksums)

**Files to create/modify:**
- `manifests/b/BOIFamily/BOICLI/0.1.0.yaml` -- local copy
- PR to `microsoft/winget-pkgs`

---

### I-10: Scoop Bucket

**What:** JSON manifest for installation via `scoop install boi`.

**Why:** Popular package manager for Windows developers. Lower friction than manual download. Part of Tier 2 strategy.

**Manifest location:** Separate repo `boi-family/scoop-bucket` with `bucket/boi.json`.

```json
{
  "version": "0.1.0",
  "description": "BOI CLI -- AI Agent Runtime for the BOI Family",
  "homepage": "https://github.com/wersoul-source/BOI-CLI",
  "license": "MIT",
  "architecture": {
    "64bit": {
      "url": "https://github.com/wersoul-source/BOI-CLI/releases/download/v0.1.0/boi_Windows_amd64.zip",
      "hash": "sha256:PLACEHOLDER"
    },
    "arm64": {
      "url": "https://github.com/wersoul-source/BOI-CLI/releases/download/v0.1.0/boi_Windows_arm64.zip",
      "hash": "sha256:PLACEHOLDER"
    }
  },
  "bin": "boi.exe"
}
```

**What to build:**
1. Create `boi-family/scoop-bucket` GitHub repo
2. Create `bucket/boi.json` with populated hashes
3. Test: `scoop bucket add boi-family https://github.com/boi-family/scoop-bucket`
4. Test: `scoop install boi`
5. Document in README

**Effort:** 1 hour

**Dependencies:** I-6, I-7

**Files to create:**
- New repo: `boi-family/scoop-bucket`
- `bucket/boi.json`

---

### I-11: `boi version` Subcommand Fix

**Current behavior:** `boi --version` works via Cobra built-in, but `boi version` is not a registered subcommand.

**Why:** Users expect both `--version` and `version` to work. Currently only `--version` works because there's no `version` subcommand registered.

**What to build:**
1. Create `internal/cli/version.go` with a `versionCmd`
2. Register on `rootCmd`
3. Embed version, commit, and build date via `-ldflags`

**Implementation:**
```go
var (
    Version   = "0.1.0"  // set via -ldflags
    Commit    = "unknown"
    BuildDate = "unknown"
)

var versionCmd = &cobra.Command{
    Use:   "version",
    Short: "Print version information",
    Run: func(cmd *cobra.Command, args []string) {
        fmt.Printf("boi version %s\n", Version)
        fmt.Printf("  commit: %s\n", Commit)
        fmt.Printf("  built:  %s\n", BuildDate)
    },
}
```

**Makefile update:**
```makefile
LDFLAGS = -X github.com/boi-family/boi-cli/internal/cli.Version=$(VERSION) \
          -X github.com/boi-family/boi-cli/internal/cli.Commit=$(shell git rev-parse --short HEAD) \
          -X github.com/boi-family/boi-cli/internal/cli.BuildDate=$(shell date -u +%Y-%m-%dT%H:%M:%SZ)

build:
    go build -ldflags "$(LDFLAGS)" -o $(BUILD_DIR)/$(BINARY_NAME) ./cmd/boi
```

**Effort:** 1 hour

**Dependencies:** None

**Files to create/modify:**
- `internal/cli/version.go` -- new file
- `internal/cli/root.go` -- register version command
- `Makefile` -- add LDFLAGS

---

### I-12: `.env` Auto-Detect + Prompt

**What:** On first run (or when no providers are configured), detect if `.env` exists and prompt user to set up providers interactively.

**Why:** Reduces the biggest friction point: users must manually `cp .env.example .env` and edit it. An interactive setup makes this 10x smoother.

**Flow:**
```
No PSC_* env vars detected
    |
    v
Check: .env file exists?
    |-> YES -> load it -> check if valid -> use it
    |-> NO  -> prompt: "Set up AI provider? [Y/n]"
                  |
                  v
              Choose provider:
              [1] OpenAI (gpt-4.1-mini)
              [2] Anthropic (claude-sonnet-5)
              [3] Local (Ollama)
              [4] Custom (any OpenAI-compatible)
              [5] Skip -- use simulated mode
                  |
                  v
              Enter API key: ****
              Enter model [default]: ...
              Enter base URL [default]: ...
                  |
                  v
              Save to .env
              Test connection
              Success / Failure
```

**What to build:**
1. `internal/cli/setup.go` -- interactive provider setup
2. Trigger on first `boi ask` or `boi` TUI launch when no providers
3. Save settings to `.env` file with proper format
4. Mask API key input (don't echo to terminal)
5. Test connection before saving

**Effort:** 3 hours

**Dependencies:** I-4 (doctor verifies provider status)

**Files to create/modify:**
- `internal/cli/setup.go` -- new file
- `internal/cli/agent.go` -- trigger setup on missing providers
- `internal/tui/splash.go` -- trigger setup from TUI

---

### I-13: CI/CD Pipeline (GitHub Actions)

**What:** Automated CI/CD pipeline that runs on every push and PR. Tests, lints, builds, and optionally releases.

**Why:** Catch regressions early. Automate release process. Standard for any production Go project.

**Pipeline jobs:**

```
.github/workflows/ci.yml (on push, PR)
    |-- test:      go test ./... -race -cover
    |-- lint:      go vet ./...
    |-- build:     go build ./cmd/boi (all platforms matrix)
    |-- security:  gosec, govulncheck

.github/workflows/release.yml (on tag v*)
    |-- goreleaser: build + archive + checksum + publish
    |-- homebrew:   update formula repo with new version + hash
    |-- winget:     submit PR to winget-pkgs
    |-- scoop:      update bucket manifest
```

**CI workflow:**
```yaml
name: CI

on:
  push:
    branches: [main]
  pull_request:
    branches: [main]

jobs:
  test:
    runs-on: ubuntu-latest
    steps:
      - uses: actions/checkout@v4
      - uses: actions/setup-go@v5
        with:
          go-version: '1.24'
      - run: go test ./... -race -coverprofile=coverage.out
      - run: go tool cover -func=coverage.out

  lint:
    runs-on: ubuntu-latest
    steps:
      - uses: actions/checkout@v4
      - uses: actions/setup-go@v5
        with:
          go-version: '1.24'
      - run: go vet ./...
      - uses: golangci/golangci-lint-action@v4

  build-matrix:
    runs-on: ubuntu-latest
    strategy:
      matrix:
        goos: [windows, darwin, linux]
        goarch: [amd64, arm64]
    steps:
      - uses: actions/checkout@v4
      - uses: actions/setup-go@v5
        with:
          go-version: '1.24'
      - run: GOOS=${{ matrix.goos }} GOARCH=${{ matrix.goarch }} go build ./cmd/boi
```

**Release workflow:**
```yaml
name: Release

on:
  push:
    tags: ['v*']

jobs:
  goreleaser:
    runs-on: ubuntu-latest
    permissions:
      contents: write
    steps:
      - uses: actions/checkout@v4
      - uses: actions/setup-go@v5
        with:
          go-version: '1.24'
      - uses: goreleaser/goreleaser-action@v5
        with:
          args: release --clean
        env:
          GITHUB_TOKEN: ${{ secrets.GITHUB_TOKEN }}
```

**Effort:** 3 hours

**Dependencies:** I-14 (cross-compile must work in CI), I-6 (goreleaser config)

**Files to create:**
- `.github/workflows/ci.yml`
- `.github/workflows/release.yml`

---

### I-14: Cross-Platform Build Setup

**What:** Ensure `go build` produces correct binaries for all 6 target platforms (Windows/Darwin/Linux x amd64/arm64).

**Why:** Foundation for everything else. Without cross-compile, no releases, no installers, no CI.

**Target matrix:**
| OS | Arch | Binary Name | Archive |
|----|------|-------------|---------|
| Windows | amd64 | `boi.exe` | `boi_Windows_amd64.zip` |
| Windows | arm64 | `boi.exe` | `boi_Windows_arm64.zip` |
| Darwin | amd64 | `boi` | `boi_Darwin_amd64.tar.gz` |
| Darwin | arm64 | `boi` | `boi_Darwin_arm64.tar.gz` |
| Linux | amd64 | `boi` | `boi_Linux_amd64.tar.gz` |
| Linux | arm64 | `boi` | `boi_Linux_arm64.tar.gz` |

**What to build:**
1. Add `release` target to Makefile
2. Verify each binary starts correctly on target platform
3. Verify no dynamic library dependencies (static linking)
4. Verify file size < 10 MB per binary

**Makefile update:**
```makefile
LDFLAGS = -s -w -X main.version=$(VERSION)

build-all:
    GOOS=windows GOARCH=amd64 go build -ldflags "$(LDFLAGS)" -o bin/boi-windows-amd64.exe ./cmd/boi
    GOOS=windows GOARCH=arm64 go build -ldflags "$(LDFLAGS)" -o bin/boi-windows-arm64.exe ./cmd/boi
    GOOS=darwin  GOARCH=amd64 go build -ldflags "$(LDFLAGS)" -o bin/boi-darwin-amd64 ./cmd/boi
    GOOS=darwin  GOARCH=arm64 go build -ldflags "$(LDFLAGS)" -o bin/boi-darwin-arm64 ./cmd/boi
    GOOS=linux   GOARCH=amd64 go build -ldflags "$(LDFLAGS)" -o bin/boi-linux-amd64 ./cmd/boi
    GOOS=linux   GOARCH=arm64 go build -ldflags "$(LDFLAGS)" -o bin/boi-linux-arm64 ./cmd/boi
```

**Effort:** 2 hours

**Dependencies:** None (foundation task)

**Files to modify:**
- `Makefile` -- add release/build-all targets
- `.gitignore` -- add bin/ entries

---

## Priority Roadmap

```
WEEK 1  (RED: Must have for v0.2.0)
├── I-14  Cross-platform build (foundation)
├── I-11  boi version subcommand
├── I-13  CI/CD pipeline
├── I-6   GitHub Releases + goreleaser
├── I-7   Checksum generation
├── I-1   install.ps1 finalize
└── I-2   install.sh finalize

WEEK 2  (YELLOW: Should have for v0.2.0)
├── I-3   boi upgrade command
├── I-4   boi doctor command
├── I-5   boi uninstall command
├── I-12  .env auto-detect + prompt
├── I-8   Homebrew formula
└── I-9   WinGet manifest

WEEK 3  (GREEN: Nice to have)
└── I-10  Scoop bucket
```

---

## Success Criteria for v0.2.0

- [ ] User can `irm https://boi.sh/install.ps1 | iex` and get working BOI CLI on Windows
- [ ] User can `curl -fsSL https://boi.sh/install.sh | bash` and get working BOI CLI on macOS/Linux
- [ ] User can `boi upgrade` to update without reinstalling
- [ ] User can `boi doctor` to diagnose any issue
- [ ] User can `brew install boi-family/boi-cli` on macOS
- [ ] User can `winget install BOIFamily.BOICLI` on Windows
- [ ] GitHub Actions CI passes on every push
- [ ] goreleaser produces release on every `v*` tag
- [ ] All 6 platform binaries pass basic smoke tests
- [ ] SHA256SUMS.txt published with every release

---

## Related Documents

- `docs/INSTALLER_FLOW.md` -- Detailed installer flow design
- `docs/INSTALL_PLAN.md` -- Install strategy from 13-tool study
- `docs/FIRST_RUN.md` -- First run experience design
- `docs/RELEASE_CHECKLIST.md` -- Per-release checklist
- `docs/CLI_COMMANDS.md` -- Full command reference
- `docs/SYSTEM_FLOW.md` -- System operational flow
- `docs/CLI_ARCHITECTURE.md` -- Internal architecture

---

*End of BUILD_PLAN.md*
