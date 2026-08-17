# 🌊 BOI CLI — Installer Flow

> ออกแบบ: 31 กรกฎาคม 2026
> โดย: Kampun (คำปัน) — BOI Family
> อ้างอิง: INSTALL_PLAN.md

---

## User Journey: From Discovery to Ready

```
User discovers BOI CLI
    │
    ├── github.com/wersoul-source/BOI-CLI → README
    ├── Word of mouth
    ├── Homebrew search
    └── WinGet search
         │
         ▼
┌─────────────────────────────────────────────────────────────────────────────┐
│                          CHOOSE INSTALL METHOD                              │
├─────────────────────────────────────────────────────────────────────────────┤
│                                                                             │
│  🪟  Windows         🍎  macOS             🐧  Linux                       │
│  ───────────         ─────────             ─────────                       │
│  1️⃣ irm | iex         1️⃣ curl | bash        1️⃣ curl | bash                 │
│  2️⃣ winget             2️⃣ homebrew            2️⃣ go install                 │
│  3️⃣ scoop              3️⃣ go install          3️⃣ apt (future)               │
│  4️⃣ go install         4️⃣ manual              4️⃣ docker                     │
│  5️⃣ manual                                                                  │
│                                                                             │
└─────────────────────────────────────────────────────────────────────────────┘
```

---

## Flow 1: One-Liner Script (`install.ps1` / `install.sh`)

```
User runs:
  irm https://boi.sh/install.ps1 | iex
  curl -fsSL https://boi.sh/install.sh | bash
         │
         ▼
┌──────────────────────────────────────────────────────────────────┐
│  STEP 1: DETECT ENVIRONMENT                                      │
│                                                                  │
│  OS:     Windows / macOS / Linux                                 │
│  Arch:   amd64 / arm64                                           │
│  Shell:  PowerShell / bash / zsh                                 │
│  Go:     check if go installed (for go install users)            │
└──────────────────────┬───────────────────────────────────────────┘
                       ▼
┌──────────────────────────────────────────────────────────────────┐
│  STEP 2: RESOLVE LATEST VERSION                                  │
│                                                                  │
│  Fetch:  https://api.github.com/repos/wersoul-source/BOI-CLI/    │
│          releases/latest                                         │
│  Parse:  tag_name → e.g. "v0.1.0"                                │
│  Build:  download URL from assets                                │
│          boi_Windows_amd64.zip                                   │
│          boi_Darwin_amd64.tar.gz                                 │
│          boi_Darwin_arm64.tar.gz                                 │
│          boi_Linux_amd64.tar.gz                                  │
│          boi_Linux_arm64.tar.gz                                  │
└──────────────────────┬───────────────────────────────────────────┘
                       ▼
┌──────────────────────────────────────────────────────────────────┐
│  STEP 3: CREATE INSTALL DIRECTORIES                              │
│                                                                  │
│  Windows:  $env:USERPROFILE\.boi\bin\                            │
│  macOS:    ~/.boi/bin/                                           │
│  Linux:    ~/.boi/bin/                                           │
│                                                                  │
│  Also:     $env:USERPROFILE\.boi\bin\boi.old  (backup version)   │
└──────────────────────┬───────────────────────────────────────────┘
                       ▼
┌──────────────────────────────────────────────────────────────────┐
│  STEP 4: DOWNLOAD BINARY                                         │
│                                                                  │
│  Download from GitHub Releases asset URL                          │
│  Show progress bar (optional)                                    │
│  Extract archive (.zip / .tar.gz)                                │
│  Place binary: ~/.boi/bin/boi (or boi.exe on Windows)            │
│  Set executable permissions (chmod +x on macOS/Linux)            │
└──────────────────────┬───────────────────────────────────────────┘
                       ▼
┌──────────────────────────────────────────────────────────────────┐
│  STEP 5: VERIFY CHECKSUM                                         │
│                                                                  │
│  Download: SHA256SUMS.txt from same release                      │
│  Compute:  sha256sum ~/.boi/bin/boi                              │
│  Compare:  if mismatch → abort, show warning                     │
│  If pass:  ✅ Checksum verified                                   │
│                                                                  │
│  (Future) SLSA attestation verification via gh CLI               │
└──────────────────────┬───────────────────────────────────────────┘
                       ▼
┌──────────────────────────────────────────────────────────────────┐
│  STEP 6: ADD TO PATH                                             │
│                                                                  │
│  Windows:                                                        │
│    [Environment]::SetEnvironmentVariable(                        │
│      'Path',                                                     │
│      "$env:USERPROFILE\.boi\bin;" +                             │
│      [Environment]::GetEnvironmentVariable('Path', 'User'),     │
│      'User'                                                      │
│    )                                                             │
│                                                                  │
│  macOS / Linux:                                                  │
│    Detect shell: bash / zsh / fish                               │
│    Append to profile:                                            │
│      export PATH="$HOME/.boi/bin:$PATH"                          │
│    Files updated:                                                │
│      ~/.bashrc, ~/.zshrc, ~/.profile, ~/.config/fish/config.fish│
└──────────────────────┬───────────────────────────────────────────┘
                       ▼
┌──────────────────────────────────────────────────────────────────┐
│  STEP 7: INITIALIZE                                              │
│                                                                  │
│  Run:    boi init --silent                                       │
│  Result: .boi/ created with default config + personas            │
│                                                                  │
│  (silent mode = no prompts, use defaults)                        │
└──────────────────────┬───────────────────────────────────────────┘
                       ▼
┌──────────────────────────────────────────────────────────────────┐
│  STEP 8: SUCCESS MESSAGE                                         │
│                                                                  │
│  ┌──────────────────────────────────────────┐                    │
│  │                                          │                    │
│  │   ✅ BOI CLI installed successfully!     │                    │
│  │                                          │                    │
│  │   Version:  v0.1.0                       │                    │
│  │   Binary:   ~/.boi/bin/boi               │                    │
│  │                                          │                    │
│  │   Quick start:                           │                    │
│  │     boi              → Launch TUI        │                    │
│  │     boi ask "hello"  → Test AI           │                    │
│  │     boi doctor       → Health check      │                    │
│  │                                          │                    │
│  │   Next: Set up PSC providers in .env     │                    │
│  │     cp .env.example .env                 │                    │
│  │                                          │                    │
│  │   Docs: https://boi.sh/docs              │                    │
│  │                                          │                    │
│  └──────────────────────────────────────────┘                    │
└──────────────────────────────────────────────────────────────────┘
```

---

## Flow 2: Homebrew (`brew install`)

```
User runs: brew install boi-family/boi-cli
         │
         ▼
┌──────────────────────────────────────────────────────────────┐
│  Homebrew Core Process                                        │
│                                                              │
│  1. tap "boi-family/boi-cli" → clone formula repo            │
│  2. read Formula/boi.rb                                      │
│  3. verify SHA256 from formula                                │
│  4. download binary from GitHub Releases                      │
│  5. extract to Homebrew prefix                                │
│  6. symlink: /usr/local/bin/boi → Cellar/boi-cli/v0.1.0/boi  │
│  7. done!                                                     │
│                                                              │
│  Update: brew upgrade boi-cli                                │
│  Uninstall: brew uninstall boi-cli                           │
└──────────────────────────────────────────────────────────────┘
```

### Homebrew Formula: `Formula/boi.rb`

```ruby
class Boi < Formula
  desc "BOI CLI — AI Agent Runtime for the BOI Family"
  homepage "https://github.com/wersoul-source/BOI-CLI"
  version "0.1.0"
  license "MIT"

  on_macos do
    if Hardware::CPU.arm?
      url "https://github.com/wersoul-source/BOI-CLI/releases/download/v0.1.0/boi_Darwin_arm64.tar.gz"
      sha256 "PLACEHOLDER_SHA256_ARM64"
    else
      url "https://github.com/wersoul-source/BOI-CLI/releases/download/v0.1.0/boi_Darwin_amd64.tar.gz"
      sha256 "PLACEHOLDER_SHA256_AMD64"
    end
  end

  on_linux do
    if Hardware::CPU.arm?
      url "https://github.com/wersoul-source/BOI-CLI/releases/download/v0.1.0/boi_Linux_arm64.tar.gz"
      sha256 "PLACEHOLDER_SHA256_LINUX_ARM64"
    else
      url "https://github.com/wersoul-source/BOI-CLI/releases/download/v0.1.0/boi_Linux_amd64.tar.gz"
      sha256 "PLACEHOLDER_SHA256_LINUX_AMD64"
    end
  end

  def install
    bin.install "boi"
  end

  test do
    system "#{bin}/boi", "version"
  end
end
```

---

## Flow 3: WinGet (`winget install`)

```
User runs: winget install BOIFamily.BOICLI
         │
         ▼
┌──────────────────────────────────────────────────────────────┐
│  WinGet Process                                               │
│                                                              │
│  1. search winget-pkgs for BOIFamily.BOICLI                   │
│  2. read YAML manifest                                       │
│  3. download installer from GitHub Releases                   │
│  4. verify SHA256 from manifest                               │
│  5. run installer / extract binary                            │
│  6. done!                                                    │
│                                                              │
│  Update: winget upgrade BOIFamily.BOICLI                      │
│  Uninstall: winget uninstall BOIFamily.BOICLI                │
└──────────────────────────────────────────────────────────────┘
```

### WinGet Manifest: `manifests/b/BOIFamily/BOICLI/0.1.0.yaml`

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

---

## Flow 4: Go Install

```
User runs: go install github.com/boi-family/boi-cli/cmd/boi@latest
         │
         ▼
┌──────────────────────────────────────────────────────────────┐
│  Go Toolchain Process                                         │
│                                                              │
│  1. resolve module @latest → version from go.sum DB           │
│  2. verify checksum via sum.golang.org                        │
│  3. download source                                          │
│  4. compile native binary                                    │
│  5. output to $GOPATH/bin/boi                                │
│  6. done!                                                    │
│                                                              │
│  Update: go install github.com/boi-family/boi-cli/cmd/boi@v0.2.0│
│  Uninstall: rm $(go env GOPATH)/bin/boi                      │
└──────────────────────────────────────────────────────────────┘
```

---

## Flow 5: Scoop

```
User runs: scoop bucket add boi-family https://github.com/boi-family/scoop-bucket
           scoop install boi
         │
         ▼
┌──────────────────────────────────────────────────────────────┐
│  Scoop Process                                                │
│                                                              │
│  1. read boi.json from bucket                                │
│  2. download archive from GitHub Releases                     │
│  3. verify hash from manifest                                 │
│  4. extract to ~/scoop/apps/boi/0.1.0/                        │
│  5. shim: ~/scoop/shims/boi.ps1 → actual binary              │
│  6. done!                                                    │
│                                                              │
│  Update: scoop update boi                                    │
│  Uninstall: scoop uninstall boi                              │
└──────────────────────────────────────────────────────────────┘
```

### Scoop Manifest: `bucket/boi.json`

```json
{
  "version": "0.1.0",
  "description": "BOI CLI — AI Agent Runtime",
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

---

## Flow 6: Manual (GitHub Releases)

```
User visits: https://github.com/wersoul-source/BOI-CLI/releases
         │
         ▼
┌──────────────────────────────────────────────────────────────┐
│  Manual Download & Install                                    │
│                                                              │
│  1. Choose asset:                                            │
│       boi_Windows_amd64.zip                                  │
│       boi_Darwin_amd64.tar.gz                                │
│       boi_Darwin_arm64.tar.gz                                │
│       boi_Linux_amd64.tar.gz                                 │
│       boi_Linux_arm64.tar.gz                                 │
│                                                              │
│  2. Download + extract                                       │
│  3. Move boi to a directory in PATH                           │
│  4. Optional: verify SHA256 against SHA256SUMS.txt            │
│  5. Run: boi init                                            │
│  6. done!                                                    │
└──────────────────────────────────────────────────────────────┘
```

---

## Flow 7: Future — Docker

```
User runs: docker pull ghcr.io/boi-family/boi-cli:latest
           docker run -it -v $(pwd):/workspace ghcr.io/boi-family/boi-cli
         │
         ▼
┌──────────────────────────────────────────────────────────────┐
│  Docker Container                                             │
│                                                              │
│  FROM golang:1.24-alpine AS build                            │
│  FROM alpine:latest                                          │
│  COPY --from=build /app/boi /usr/local/bin/boi               │
│  ENTRYPOINT ["boi"]                                          │
│                                                              │
│  Use case: CI/CD, isolated environments                      │
│  Not primary distribution method                             │
└──────────────────────────────────────────────────────────────┘
```

---

## Flow 8: Update (`boi upgrade`)

```
User runs: boi upgrade
         │
         ▼
┌──────────────────────────────────────────────────────────────┐
│  STEP 1: CHECK CURRENT VERSION                                │
│                                                              │
│  Current:  v0.1.0 (embedded via -ldflags at build)           │
└──────────────────────┬───────────────────────────────────────┘
                       ▼
┌──────────────────────────────────────────────────────────────┐
│  STEP 2: FETCH LATEST RELEASE                                 │
│                                                              │
│  GET https://api.github.com/repos/wersoul-source/BOI-CLI/    │
│      releases/latest                                         │
│                                                              │
│  Parse:  tag_name → "v0.2.0"                                 │
└──────────────────────┬───────────────────────────────────────┘
                       ▼
┌──────────────────────────────────────────────────────────────┐
│  STEP 3: COMPARE                                              │
│                                                              │
│  If same:    "Already up to date (v0.1.0)"                   │
│  If older:   "Current v0.1.0 → Latest v0.2.0"                │
│              "Update? [Y/n]"                                  │
└──────────────────────┬───────────────────────────────────────┘
                       ▼
┌──────────────────────────────────────────────────────────────┐
│  STEP 4: DOWNLOAD + BACKUP                                   │
│                                                              │
│  Backup:  cp ~/.boi/bin/boi ~/.boi/bin/boi.old               │
│  Download new binary to ~/.boi/bin/boi-new                    │
│  Verify SHA256 against SHA256SUMS.txt                         │
└──────────────────────┬───────────────────────────────────────┘
                       ▼
┌──────────────────────────────────────────────────────────────┐
│  STEP 5: REPLACE                                              │
│                                                              │
│  If checksum OK:                                             │
│    mv ~/.boi/bin/boi-new ~/.boi/bin/boi                      │
│    chmod +x ~/.boi/bin/boi                                   │
│  If checksum FAIL:                                           │
│    rm ~/.boi/bin/boi-new                                     │
│    "Checksum verification failed. Aborting."                  │
│    Keep current version                                       │
└──────────────────────┬───────────────────────────────────────┘
                       ▼
┌──────────────────────────────────────────────────────────────┐
│  STEP 6: VERIFY                                               │
│                                                              │
│  Run: boi version → v0.2.0                                   │
│  Success: "Updated to v0.2.0"                                │
│  Note:    "Previous version saved as boi.old"                 │
└──────────────────────────────────────────────────────────────┘
```

---

## Complete Installer Architecture

```
                    ┌──────────────────────────────────┐
                    │     github.com/wersoul-source    │
                    │           /BOI-CLI               │
                    │                                  │
                    │  ┌─ boi_Windows_amd64.zip        │
                    │  ├─ boi_Darwin_amd64.tar.gz      │
                    │  ├─ boi_Darwin_arm64.tar.gz      │
                    │  ├─ boi_Linux_amd64.tar.gz       │
                    │  ├─ boi_Linux_arm64.tar.gz       │
                    │  ├─ SHA256SUMS.txt               │
                    │  └─ Source code (.zip/.tar.gz)   │
                    └──────┬──────────┬────────────────┘
                           │          │
              ┌────────────┘          └────────────┐
              ▼                                    ▼
    ┌─────────────────────┐            ┌─────────────────────┐
    │  Package Managers    │            │  Direct Install      │
    │                     │            │                     │
    │  Homebrew Formula    │            │  install.ps1         │
    │  WinGet Manifest     │            │  install.sh          │
    │  Scoop Bucket        │            │                     │
    │  Go Install          │            │  curl | bash         │
    │                     │            │  irm | iex            │
    │  brew install        │            │                     │
    │  winget install      │            │  Manual Download     │
    │  scoop install       │            │                     │
    │  go install          │            │  boi upgrade         │
    └──────────┬───────────┘            └──────────┬──────────┘
               │                                   │
               └───────────────┬───────────────────┘
                               ▼
                    ┌─────────────────────┐
                    │    ~/.boi/bin/boi    │
                    │                     │
                    │  First Run:          │
                    │    boi init          │
                    │    boi doctor        │
                    │    boi               │
                    └─────────────────────┘
```

---

*สิ้นสุด INSTALLER_FLOW.md*
