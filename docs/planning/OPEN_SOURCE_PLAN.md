# BOI CLI — Open Source Release & Community Plan

## Roadmap

### v0.1.0 — Initial Release (now)

- [x] MIT License
- [x] README complete (features, personas, quick start, FAQ)
- [x] `boi upgrade` working (fetches from GitHub Releases API)
- [x] Cross-platform binaries on GitHub Releases (Windows, Linux, macOS × amd64, arm64)
- [x] CI/CD with GitHub Actions + GoReleaser
- [x] CHANGELOG.md
- [x] Makefile (build, test, lint, install)

### v0.2.0 — Community Ready

- [x] CONTRIBUTING.md
- [x] Issue templates (bug report, feature request)
- [x] PR template
- [x] CODE_OF_CONDUCT.md (Contributor Covenant v2.1)
- [x] SECURITY.md
- [ ] `boi upgrade` with checksum verification
- [ ] `boi upgrade` with `--check` flag
- [ ] Discord / community channel
- [ ] GitHub Discussions enabled

### v0.3.0 — Skill Ecosystem

- [ ] Community-contributed skills
- [ ] Skill registry / marketplace
- [ ] `boi skill install <name>` from registry
- [ ] Skill authoring guide

### v1.0.0 — Stable Release

- [ ] All 6 personas fully operational
- [ ] Documentation complete (CLI, TUI, skills, personas)
- [ ] Package managers:
  - [ ] Homebrew (`brew install boi`)
  - [ ] WinGet (`winget install boi`)
  - [ ] Scoop (`scoop install boi`)
  - [ ] Arch AUR
  - [ ] Nix
- [ ] Community maintainers
- [ ] Backward compatibility commitment
- [ ] Semver from 1.0.0

---

## Upgrade Flow

```
User runs: boi upgrade
    │
    ▼
Check GitHub Releases API
  GET https://api.github.com/repos/boi-family/boi-cli/releases/latest
    │
    ├── No network → "Failed to check for updates"
    │
    ├── Latest === current → "Already up to date."
    │
    └── Latest > current → Download binary
         │
         ├── Download boi_<version>_<os>_<arch>.tar.gz
         ├── Verify SHA256 checksum against checksums.txt    [v0.2.0]
         ├── Extract archive
         ├── Verify binary exists and is non-empty
         ├── Rename current → .old (rollback safety)
         ├── Copy new binary in place
         ├── Remove .old
         └── Restart with new version
```

### Checksum Verification (v0.2.0)

When `boi upgrade` downloads a binary, it fetches `checksums.txt` from
the release and verifies the SHA256 hash before replacing the binary.

### Rollback

If the upgrade fails mid-process, the `.old` backup is restored.
Users can also manually restore from `.old` or re-download.

### Pre-release Blocks

`boi upgrade` only upgrades to stable releases (tags matching `v*` without
`-alpha`, `-beta`, `-rc` suffix). Pre-release participation is opt-in via
`boi upgrade --prerelease`.

---

## Release Process

1. Update `Version` in `internal/cli/version.go`
2. Update `CHANGELOG.md`
3. Commit: `git commit -m "chore: bump to vX.Y.Z"`
4. Tag: `git tag vX.Y.Z`
5. Push: `git push && git push --tags`
6. GitHub Actions + GoReleaser builds and uploads:
   - 6 binary archives (3 OS × 2 arch)
   - `checksums.txt`
   - Source tarball
   - Changelog from commit history

---

## Community Channels

| Channel | Purpose |
|---------|---------|
| GitHub Issues | Bug reports, feature requests |
| GitHub Discussions | Q&A, ideas, showcase |
| Discord (coming v0.2.0) | Real-time chat, community support |
| Email `boi-family@proton.me` | Security reports, private inquiries |

---

## Governance

- **Benevolent Dictator**: Kampun (คำปัน) — final decision on direction and scope
- **Maintainers**: BOI Family personas (Boi, Dang, Don, Kine, Kamkaew)
- **Contributors**: Community members with merged PRs

All decisions are made in public GitHub issues and discussions.

---

*Created: July 2026 — Kampun (คำปัน)*
