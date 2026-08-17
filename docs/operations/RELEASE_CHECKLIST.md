# ✅ BOI CLI — Release Checklist

> ใช้ก่อนทุก release
> เวอร์ชันล่าสุด: v0.1.0

---

## Pre-Release Checklist

### Code Quality

- [ ] All tests pass: `go test ./...`
- [ ] Lint passes: `go vet ./...`
- [ ] Build succeeds: `go build -o bin/boi ./cmd/boi`
- [ ] No `TODO` or `FIXME` left in production code
- [ ] No hardcoded paths or secrets
- [ ] All error messages are user-friendly (no stack traces in CLI output)

### Version Bump

- [ ] Version in `internal/transport/cli/version.go` (`Version: "x.y.z"`)
- [ ] Version in `README.md` badges
- [ ] Version tag created: `git tag -a vx.y.z -m "vx.y.z: Release description"`
- [ ] Tag follows semver: `v<major>.<minor>.<patch>`

### Documentation

- [ ] `README.md` updated (if new features)
- [ ] `GET_STARTED.md` updated (if install flow changed)
- [ ] `docs/reference/CLI_COMMANDS.md` updated (if commands changed)
- [ ] `CHANGELOG.md` updated with all changes
- [ ] Breaking changes documented with migration guide

### Changelog Format

```markdown
# Changelog

## [0.1.0] - 2026-07-31

### Added
- Core CLI with Cobra commands
- PSC (Provider Supply Chain) with 4 providers
- Persona system with 6 personas
- Skill system with built-in skills
- Phantom DB memory with FTS5 search
- Agent loop with ReAct pattern
- Bubbletea TUI interface

### Changed
- (if any)

### Fixed
- (if any)

### Removed
- (if any)

### Security
- (if any)
```

### Binary Build

- [ ] Windows amd64: `GOOS=windows GOARCH=amd64 go build -o bin/boi.exe ./cmd/boi`
- [ ] Windows arm64: `GOOS=windows GOARCH=arm64 go build -o bin/boi.exe ./cmd/boi`
- [ ] macOS amd64: `GOOS=darwin GOARCH=amd64 go build -o bin/boi-darwin-amd64 ./cmd/boi`
- [ ] macOS arm64: `GOOS=darwin GOARCH=arm64 go build -o bin/boi-darwin-arm64 ./cmd/boi`
- [ ] Linux amd64: `GOOS=linux GOARCH=amd64 go build -o bin/boi-linux-amd64 ./cmd/boi`
- [ ] Linux arm64: `GOOS=linux GOARCH=arm64 go build -o bin/boi-linux-arm64 ./cmd/boi`
- [ ] All binaries are statically linked (no DLL dependencies)

### Binary Size Check

| Platform | Target | Actual | OK? |
|----------|--------|--------|-----|
| Windows amd64 | < 10 MB | ___ | |
| Windows arm64 | < 10 MB | ___ | |
| macOS amd64 | < 10 MB | ___ | |
| macOS arm64 | < 10 MB | ___ | |
| Linux amd64 | < 10 MB | ___ | |
| Linux arm64 | < 10 MB | ___ | |

### Archive & Checksums

- [ ] Create archives for each binary:
  ```bash
  zip boi_Windows_amd64.zip boi.exe
  tar -czf boi_Darwin_amd64.tar.gz boi-darwin-amd64
  tar -czf boi_Darwin_arm64.tar.gz boi-darwin-arm64
  tar -czf boi_Linux_amd64.tar.gz boi-linux-amd64
  tar -czf boi_Linux_arm64.tar.gz boi-linux-arm64
  ```
- [ ] Generate SHA256 checksums: `sha256sum bin/* > SHA256SUMS.txt`
- [ ] Verify checksums match actual files

### GitHub Release

- [ ] Create release at: `https://github.com/wersoul-source/BOI-CLI/releases/new`
- [ ] Tag: `vx.y.z`
- [ ] Title: `BOI CLI vx.y.z`
- [ ] Description: Copy from CHANGELOG.md
- [ ] Upload all 5 archives
- [ ] Upload `SHA256SUMS.txt`
- [ ] Upload source code archives (`.zip` + `.tar.gz`)
- [ ] Mark as latest release

### Install Scripts

- [ ] Update download URLs in `scripts/install.ps1` (if version-specific)
- [ ] Update download URLs in `scripts/install.sh`
- [ ] Test `install.ps1` on Windows (fresh VM or clean PATH)
- [ ] Test `install.sh` on macOS (if available)
- [ ] Test `install.sh` on Linux (Docker container)
- [ ] Test `go install github.com/boi-family/boi-cli/cmd/boi@latest`

### Homebrew Formula

- [ ] Update `Formula/boi.rb` in `homebrew-boi-cli` repo
- [ ] Update version number
- [ ] Update SHA256 hashes for all platforms
- [ ] Test: `brew install boi-family/boi-cli`
- [ ] Test: `brew test boi-cli`
- [ ] Test: `brew audit --strict boi-cli`

### WinGet Manifest

- [ ] Create YAML manifest: `manifests/b/BOIFamily/BOICLI/x.y.z.yaml`
- [ ] Update SHA256 hash for Windows binary
- [ ] Submit PR to `microsoft/winget-pkgs`
- [ ] Test: `winget install BOIFamily.BOICLI`

### Scoop Bucket

- [ ] Update `boi.json` in scoop bucket repo
- [ ] Update version + hash
- [ ] Test: `scoop install boi`

### Docker (Future)

- [ ] Build Docker image: `docker build -t ghcr.io/boi-family/boi-cli:x.y.z .`
- [ ] Push: `docker push ghcr.io/boi-family/boi-cli:x.y.z`
- [ ] Tag latest: `docker tag ... ghcr.io/boi-family/boi-cli:latest`
- [ ] Test: `docker run -it ghcr.io/boi-family/boi-cli:x.y.z`

### Post-Release Verification

- [ ] Download binary from GitHub Releases on each platform
- [ ] Verify SHA256 matches published checksums
- [ ] Run `boi --version` → correct version
- [ ] Run `boi init` → creates `.boi/` workspace
- [ ] Run `boi` → TUI launches without error
- [ ] Run `boi ask "hello"` → response (simulated OK if no API key)
- [ ] Check `boi --help` shows all commands
- [ ] Update `PROGRESS.md` with release notes
- [ ] Update `LEARNING_JOURNAL.md` with session notes (if applicable)

### Announce (if public release)

- [ ] Post on project discussion board
- [ ] Update website (if any)
- [ ] Notify early adopters / testers

---

## Automated Release Pipeline (GitHub Actions)

```yaml
# .github/workflows/release.yml
name: Release

on:
  push:
    tags:
      - 'v*'

jobs:
  goreleaser:
    runs-on: ubuntu-latest
    steps:
      - uses: actions/checkout@v4
      
      - uses: actions/setup-go@v5
        with:
          go-version: '1.24'
          
      - name: Run GoReleaser
        uses: goreleaser/goreleaser-action@v5
        with:
          distribution: goreleaser
          version: latest
          args: release --clean
        env:
          GITHUB_TOKEN: ${{ secrets.GITHUB_TOKEN }}
```

### GoReleaser Config (`~/.goreleaser.yml`):

```yaml
builds:
  - main: ./cmd/boi
    binary: boi
    goos:
      - windows
      - darwin
      - linux
    goarch:
      - amd64
      - arm64
    ldflags:
      - -s -w -X main.version={{.Version}}

archives:
  - format: tar.gz
    name_template: "{{ .ProjectName }}_{{ .Os }}_{{ .Arch }}"
    format_overrides:
      - goos: windows
        format: zip

checksum:
  name_template: 'SHA256SUMS.txt'

changelog:
  sort: asc
  filters:
    exclude:
      - '^docs:'
      - '^test:'
      - '^ci:'
      - Merge pull request
      - Merge branch

release:
  github:
    owner: wersoul-source
    name: BOI-CLI
```

---

## Quick Command Reference for Releaser

```bash
# 1. Make sure everything passes
go test ./...
go vet ./...

# 2. Update version in code
# (manual edit: internal/transport/cli/version.go)

# 3. Commit version bump
git add -A
git commit -m "chore: bump version to v0.2.0"

# 4. Tag
git tag -a v0.2.0 -m "v0.2.0: Installer + doctor + upgrade"

# 5. Push with tags
git push origin main --tags

# 6. Cross-compile all platforms
make release

# 7. Create GitHub Release (manual or via goreleaser)
goreleaser release --clean

# 8. Update package managers
# (Homebrew, WinGet, Scoop — manual PRs)
```

---

*สิ้นสุด RELEASE_CHECKLIST.md*
