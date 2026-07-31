# 🔧 BOI CLI — Installer Study & Plan

> ศึกษา: 31 กรกฎาคม 2026
> โดย: Kampun (คำปัน) — BOI Family
> สำหรับ: BOI CLI v0.1.0+

---

## สารบัญ

1. [Study Scope](#study-scope)
2. [Tool-by-Tool Analysis](#tool-by-tool-analysis)
3. [Comparison Matrix](#comparison-matrix)
4. [Patterns Extracted](#patterns-extracted)
5. [Decision: BOI CLI Install Strategy](#decision-boi-cli-install-strategy)

---

## Study Scope

ศึกษา 13 CLI tools/package managers เพื่อดึง pattern มาออกแบบ installer ของ BOI CLI:

| # | Tool | Category | Why Studied |
|---|------|----------|-------------|
| 1 | **GitHub CLI (gh)** | Precompiled binary | Gold standard: multi-platform, multi-manager, SLSA |
| 2 | **Claude Code** | AI CLI | Direct competitor: curl-pipe-bash, auto-update |
| 3 | **Ollama** | AI Runtime | Simple UX: single PowerShell command |
| 4 | **OpenCode** | AI CLI | DNA source: Go install + npm + prebuilt |
| 5 | **uv (Astral)** | Package Manager | Speed benchmark: Rust binary, 10ms install |
| 6 | **pnpm** | Package Manager | Node.js ecosystem benchmark |
| 7 | **bun** | JS Runtime | curl-pipe-sh + multi-manager |
| 8 | **Homebrew** | Package Manager | macOS standard: formula-based |
| 9 | **WinGet** | Package Manager | Windows standard: YAML manifest |
| 10 | **Scoop** | Package Manager | Windows dev: PowerShell, portable |
| 11 | **Cargo** | Package Manager | Rust ecosystem: source compilation |
| 12 | **Go install** | Build Tool | Go ecosystem: GOPATH/bin |
| 13 | **npm -g** | Package Manager | JS ecosystem: global install |

---

## Tool-by-Tool Analysis

### 1. GitHub CLI (`gh`)

```
Install:    GitHub Releases (.tar.gz/.zip) + Homebrew + WinGet + apt/yum/dnf + Scoop
Update:     Manual (brew upgrade / winget upgrade / gh upgrade)
Uninstall:  brew uninstall / winget uninstall / rm binary
Auto:       ❌ No native auto-update
Checksums:  ✅ SLSA provenance + checksums.txt
Config:     gh auth login → interactive OAuth
First Run:  Interactive setup wizard
Binary:     Go, ~30 MB
```

**Patterns to borrow:**
- Multi-manager distribution (brew, winget, scoop, apt, direct download)
- SLSA attestation for supply chain security
- Interactive first-run experience

### 2. Claude Code (Anthropic)

```
Install:    curl -fsSL https://claude.ai/install.sh | bash (primary)
            Homebrew, WinGet, npm -g (secondary)
Update:     ✅ Native auto-update (background daemon checks)
Uninstall:  rm /usr/local/bin/claude (manual)
Auto:       ✅ Yes — checks for updates in background
Checksums:  ❌ No published checksums
Config:     .claude/ settings, first-run login prompt
First Run:  Login via browser OAuth
Binary:     Node.js bundled
```

**Patterns to borrow:**
- Auto-update daemon (user never needs to think about updates)
- First-run login flow
- Multiple install surfaces: curl-pipe | brew | winget | npm

**Patterns to AVOID:**
- No checksums = supply chain risk
- curl-pipe-bash without verification
- Large binary (Node.js runtime bundled)

### 3. Ollama

```
Install:    irm https://ollama.com/install.ps1 | iex (Windows)
            curl -fsSL https://ollama.com/install.sh | sh (macOS/Linux)
Update:     Manual download + reinstall
Uninstall:  App uninstall (Windows) / rm + launchctl (macOS)
Auto:       ❌ No
Checksums:  ❌ No
Config:     OLLAMA_MODELS env var, system service
First Run:  ollama pull <model> — pull first model
Binary:     Go, ~150 MB (includes llama.cpp)
```

**Patterns to borrow:**
- Single-command install (lowest friction)
- System service for background operation
- `irm | iex` pattern for Windows (PowerShell-native)

**Patterns to AVOID:**
- 150 MB binary (embedding ML engine)
- No version pinning (always latest)

### 4. OpenCode

```
Install:    npm install -g @opencode/cli
            go install github.com/anomalyco/opencode@latest
            Download from GitHub Releases
Update:     npm update -g / go install / manual download
Uninstall:  npm uninstall -g / rm binary
Auto:       ❌ No
Checksums:  ❌ No (GitHub releases only)
Config:     opencode.json / .opencode/ in project
First Run:  opencode init → sets up config
Binary:     Go + Bun workspace monorepo
```

**Patterns to borrow:**
- Multiple ecosystem options (Go, npm in parallel)
- Tool-local config in project root (.opencode/)
- GitHub Releases as canonical distribution

### 5. uv (Astral)

```
Install:    curl -LsSf https://astral.sh/uv/install.sh | sh (primary)
            pip install uv / Homebrew / WinGet / cargo install uv
Update:     uv self update (native command)
Uninstall:  pip uninstall uv / brew uninstall / cargo uninstall
Auto:       ✅ uv self update
Checksums:  ❌ No
Config:     pyproject.toml, uv.toml
First Run:  No setup needed — "just works"
Binary:     Rust, ~20 MB standalone binary
```

**Patterns to borrow:**
- `self update` command — user-controlled but simple
- "Just works" first run — no config required
- Multiple ecosystem options (pip, brew, winget, cargo, direct)
- Single static binary — fast download + start

### 6. pnpm

```
Install:    npm install -g pnpm (Node.js required)
            curl -fsSL https://get.pnpm.io/install.sh | sh
            Standalone scripts (pnpm.cmd, pnpm)
Update:     npm update -g pnpm / pnpm self-update
Uninstall:  npm uninstall -g pnpm / rm binary
Auto:       ❌ No
Checksums:  ❌ No
Config:     .npmrc, pnpm-workspace.yaml
First Run:  pnpm init — creates package.json
Binary:     Node.js, ~3 MB (standalone shell wrapper)
```

**Patterns to borrow:**
- Standalone scripts option for Node.js-less install
- Lightweight (just a shell script wrapping Node)

**Patterns to AVOID:**
- Requires Node.js for primary install method

### 7. bun

```
Install:    curl -fsSL https://bun.sh/install | bash (primary)
            npm install -g bun / Homebrew / Docker
Update:     bun upgrade
Uninstall:  rm -rf ~/.bun
Auto:       ❌ No
Checksums:  ❌ No
Config:     bunfig.toml
First Run:  Just works — instant
Binary:     Zig, ~90 MB standalone
```

**Patterns to borrow:**
- `upgrade` command built-in
- Single curl command + instant results
- Profile-based PATH injection

### 8. Homebrew

```
Install:    /bin/bash -c "$(curl -fsSL https://raw.githubusercontent.com/Homebrew/install/HEAD/install.sh)"
Update:     brew update && brew upgrade
Uninstall:  brew uninstall <formula>
Auto:       ❌ No
Checksums:  ✅ Homebrew verifies SHA256 in formula
Config:     Ruby formula file (url + sha256 + dependencies)
First Run:  brew install <name> — auto-resolves deps
Binary:     Ruby formula definition
```

**Patterns to borrow:**
- Formula = single file with URL + SHA256 + deps
- Automatic dependency resolution
- User trusts Homebrew as gatekeeper

### 9. WinGet

```
Install:    winget install <id> (pre-installed on Windows 10+)
Update:     winget upgrade <id>
Uninstall:  winget uninstall <id>
Auto:       ❌ No
Checksums:  ✅ YAML manifest includes SHA256
Config:     YAML manifest in winget-pkgs repo
First Run:  Just works — downloads + installs
Binary:     YAML manifest + installer
```

**Patterns to borrow:**
- Zero-install package manager (pre-installed on Windows)
- SHA256 verification built into manifest
- Auto-resolves dependencies from manifest

### 10. Scoop

```
Install:    scoop install <name> (requires scoop itself installed)
Update:     scoop update <name>
Uninstall:  scoop uninstall <name>
Auto:       ❌ No
Checksums:  ✅ JSON manifest includes hash
Config:     JSON manifest + PowerShell install script
First Run:  Downloads portable app to ~/scoop/apps/
Binary:     Portable (no installer, just extract)
```

**Patterns to borrow:**
- Portable-first philosophy (no registry, no admin)
- JSON manifest = simple to generate
- User-controlled upgrade

### 11. Cargo

```
Install:    cargo install <crate> (Rust toolchain required)
Update:     cargo install --force <crate>
Uninstall:  cargo uninstall <crate>
Auto:       ❌ No
Checksums:  ✅ crates.io verifies SHA256
Config:     Cargo.toml
First Run:  Compiles from source → binary in ~/.cargo/bin
Binary:     Source compilation → native binary
```

**Patterns to borrow:**
- Ecosystem-standard install path (~/.cargo/bin)
- Version pinning via crate version
- Uninstall command built into toolchain

**Patterns to AVOID:**
- Requires Rust toolchain (not for GO project)
- Compilation time (30s–5min)

### 12. Go install

```
Install:    go install <module>@latest (Go toolchain required)
Update:     go install <module>@latest
Uninstall:  rm $GOPATH/bin/<binary>
Auto:       ❌ No
Checksums:  ✅ Go module checksum database (sum.golang.org)
Config:     go.mod + go.sum
First Run:  Compiles from source → $GOPATH/bin
Binary:     Native binary from source
```

**Patterns to borrow:**
- ✅ THIS IS OUR PATH — BOI CLI is written in Go
- go.sum = built-in checksum verification
- go install = one command to build + install
- GOPATH/bin = well-known location

### 13. npm install -g

```
Install:    npm install -g <package> (Node.js required)
Update:     npm update -g <package>
Uninstall:  npm uninstall -g <package>
Auto:       ❌ No
Checksums:  ✅ npm integrity (package-lock.json)
Config:     package.json
First Run:  Downloads + extracts to global node_modules
Binary:     JavaScript (requires Node.js runtime)
```

**Patterns to AVOID:**
- Requires Node.js runtime (we're Go)
- JavaScript distribution (not native binary)

---

## Comparison Matrix

| Tool | Primary Install | Update | Uninstall | Auto-Update | Checksums | Config | First Run |
|------|----------------|--------|-----------|-------------|-----------|--------|-----------|
| **gh** | GitHub Releases + brew/winget | `brew upgrade` / `winget upgrade` | `brew uninstall` / `winget uninstall` | ❌ No | ✅ SLSA attestation | `gh auth login` | Interactive OAuth setup |
| **claude** | `curl-pipe-bash` | Native auto-update daemon | `rm /usr/local/bin/claude` | ✅ Yes (daemon) | ❌ No | `.claude/` settings | Login via browser |
| **ollama** | `irm \| iex` | Manual download | Uninstall app | ❌ No | ❌ No | System service | `ollama pull model` |
| **opencode** | `go install` / `npm` / releases | `go install` / `npm update` | `rm binary` / `npm uninstall` | ❌ No | ❌ No | `.opencode/` | `opencode init` |
| **uv** | `curl-pipe-sh` / `pip` / `brew` | `uv self update` | `pip uninstall` / `rm` | ✅ `self update` | ❌ No | `pyproject.toml` | Just works |
| **pnpm** | `npm install -g` / `curl-sh` | `pnpm self-update` | `npm uninstall` / `rm` | ❌ No | ❌ No | `.npmrc` | `pnpm init` |
| **bun** | `curl-pipe-sh` / `npm` / `brew` | `bun upgrade` | `rm -rf ~/.bun` | ❌ No | ❌ No | `bunfig.toml` | Just works |
| **Homebrew** | `brew install formula` | `brew upgrade` | `brew uninstall` | ❌ No | ✅ SHA256 in formula | Ruby formula | Auto-resolves deps |
| **WinGet** | `winget install id` | `winget upgrade` | `winget uninstall` | ❌ No | ✅ SHA256 in manifest | YAML manifest | Downloads + installs |
| **Scoop** | `scoop install name` | `scoop update` | `scoop uninstall` | ❌ No | ✅ JSON hash | JSON manifest | Extracts portable |
| **Cargo** | `cargo install crate` | `cargo install --force` | `cargo uninstall` | ❌ No | ✅ crates.io SHA256 | `Cargo.toml` | Compiles from source |
| **Go install** | `go install pkg@latest` | `go install pkg@latest` | `rm $GOPATH/bin/boi` | ❌ No | ✅ go.sum DB | `go.mod` | Compiles from source |
| **npm -g** | `npm install -g pkg` | `npm update -g` | `npm uninstall -g` | ❌ No | ✅ npm integrity | `package.json` | Extracts to global |

---

## Patterns Extracted

### Pattern 1: "Tiered Install Strategy"
ทุก tool ที่ mature (gh, uv, bun, claude) ใช้ multi-surface install:
1. **Lowest friction:** curl-pipe-sh (บรรทัดเดียว)
2. **Ecosystem:** brew/winget/scoop (package manager)
3. **Developer:** go install/npm/cargo (source)

→ BOI CLI ควรมีทั้ง 3 tiers

### Pattern 2: "Checksums Are Table Stakes for Trust"
- gh: SLSA attestation (gold standard)
- Homebrew/WinGet/Scoop: SHA256 in formula/manifest
- go.sum: Built-in Go module checksum DB
- ❌ claude, ollama, uv, opencode: no checksums

→ BOI CLI: SHA256 checksums file ใน GitHub Releases + go.sum verification สำหรับ go install

### Pattern 3: "Self-Update Is Premium UX"
- uv: `uv self update` (user-triggered, simple)
- bun: `bun upgrade` (same pattern)
- claude: auto-update daemon (background, zero-friction)
- ❌ gh, opencode, ollama: manual

→ BOI CLI: `boi upgrade` command (user-triggered like uv/bun), ไม่ทำ auto-update daemon (ซับซ้อน + privacy)

### Pattern 4: "Single Binary = Best UX"
- uv: Rust binary, 20 MB, instant start
- bun: Zig binary, all-in-one
- ollama: Go binary, all-in-one (แต่ heavy)
- BOI CLI: Go binary, ~8 MB → เหนือกว่าทั้งหมดในด้าน size

→ เน้น single-binary ให้เร็วและเล็ก

### Pattern 5: "First Run Should Just Work"
- uv, bun: no config needed on first run
- claude, gh: require login/auth on first run
- BOI CLI: `boi` เปล่า → TUI เปิดเลย, `boi ask` → ทำงานได้ทันที (simulated mode)

→ First run = zero friction, fallback graceful

---

## Decision: BOI CLI Install Strategy

### Tier 1: One-Liner (Lowest Friction) 🥇

```
Windows:  irm https://boi.sh/install.ps1 | iex
macOS:    curl -fsSL https://boi.sh/install.sh | bash
Linux:    curl -fsSL https://boi.sh/install.sh | bash
```

Script ทำ:
1. Detect OS + Arch (amd64/arm64)
2. Download latest binary from GitHub Releases
3. Install to `~/.boi/bin/boi`
4. Add to PATH (~/.profile / ~/.bashrc / $PROFILE)
5. Verify SHA256 checksum
6. Run `boi init --silent`
7. Show success message + next steps

### Tier 2: Package Managers 🥈

| Manager | Command | Status |
|---------|---------|--------|
| **Homebrew** | `brew install boi-family/boi-cli` | Formula: `boi.rb` |
| **WinGet** | `winget install BOIFamily.BOICLI` | Manifest: `BOIFamily.BOICLI.yaml` |
| **Scoop** | `scoop install boi` | Bucket: `boi.json` |
| **Go install** | `go install github.com/boi-family/boi-cli/cmd/boi@latest` | Native, checksums via go.sum |

### Tier 3: Power Users 🥉

```
Manual:    Download from https://github.com/wersoul-source/BOI-CLI/releases
Build:     git clone && make build
Docker:    docker pull ghcr.io/boi-family/boi-cli
```

### Update Strategy

```
boi upgrade          # Check latest release, download binary, replace, verify
boi upgrade v0.2.0   # Pin specific version
boi upgrade --check  # Check if newer version exists (dry run)
```

Implementation:
- Compare `boi version` (embedded via `go build -ldflags`) กับ latest GitHub Release tag
- Download + replace binary atomically (write to temp, rename)
- Verify SHA256 before replacing
- Keep previous version as `boi.old` until next successful run

### Uninstall

```
# Windows
Remove-Item -Recurse ~/.boi
Remove-Item ~/.boi/bin

# macOS / Linux
rm -rf ~/.boi
```

Also: remove PATH entry from shell profile.

### First Run Experience

```
boi            → Splash screen → TUI (no config needed)
boi ask "..."  → Works immediately (simulated mode if no .env)
boi init       → Creates .boi/ workspace
boi doctor     → Health check: Go version, binary path, config, PSC, memory
```

### Checksum Strategy

1. **go.sum** (built-in): `go install` ตรวจสอบอัตโนมัติ
2. **SHA256SUMS.txt** (GitHub Releases): installer script ตรวจสอบก่อน install
3. **SLSA provenance** (future): generate via GitHub Actions for supply chain transparency

### Distribution Matrix

| Platform | Arch | Primary | Secondary | Tertiary |
|----------|------|---------|-----------|----------|
| **Windows** | amd64 | `install.ps1` (irm\|iex) | WinGet | Scoop, Manual |
| **Windows** | arm64 | `install.ps1` | WinGet | Manual |
| **macOS** | amd64 | `install.sh` (curl\|bash) | Homebrew | Go install, Manual |
| **macOS** | arm64 | `install.sh` | Homebrew | Go install, Manual |
| **Linux** | amd64 | `install.sh` (curl\|bash) | Go install, apt | Manual, Docker |
| **Linux** | arm64 | `install.sh` | Go install | Manual, Docker |

---

## Summary

| Aspect | Decision |
|--------|----------|
| **Primary install** | One-liner script (irm\|iex / curl\|bash) |
| **Package managers** | Homebrew, WinGet, Scoop, Go install |
| **Update** | `boi upgrade` (user-triggered, like uv) |
| **Auto-update** | ❌ No daemon (keep simple, respect privacy) |
| **Checksums** | SHA256SUMS.txt on GitHub Releases + go.sum |
| **Binary size** | ~8 MB (Go, single binary) |
| **Config** | `.boi/` in project root + `~/.boi/config.yaml` |
| **First run** | Zero config — `boi` opens TUI immediately |
| **Uninstall** | `rm -rf ~/.boi` + remove PATH entry |
| **CI/CD** | GitHub Actions: build + checksum + release + publish |

---

*สิ้นสุด INSTALL_PLAN.md*
