# Contributing to BOI CLI

Welcome! BOI CLI is the BOI Family's AI Agent Runtime — a single Go binary with
6 specialized personas, cross-session memory, and multi-provider fallback.

We're glad you want to contribute.

## How to Report Bugs

1. Search [existing issues](https://github.com/wersoul-source/BOI-CLI/issues) to avoid duplicates.
2. If none found, [open a bug report](https://github.com/wersoul-source/BOI-CLI/issues/new?template=bug_report.md).
3. Include:
   - `boi version` output
   - Your OS and architecture (`uname -a` / `go env GOOS GOARCH`)
   - Steps to reproduce
   - Expected vs actual behavior

## How to Suggest Features

Use the [feature request template](https://github.com/wersoul-source/BOI-CLI/issues/new?template=feature_request.md).
Describe the problem you're solving and how BOI CLI could help.

## Development Setup

```bash
git clone https://github.com/wersoul-source/BOI-CLI.git
cd BOI-CLI
go build -o bin/boi ./cmd/boi
./bin/boi init
```

**Requirements:**
- Go 1.24+
- No external dependencies — BOI CLI compiles to a single static binary

## Code Style

We follow standard Go conventions:

```bash
go fmt ./...        # Format code
go vet ./...        # Static analysis
go build ./...      # Verify compilation
go test ./...       # Run tests
```

Or use the Makefile:

```bash
make lint           # go vet
make test           # go test
make build          # cross-platform build
```

## Project Structure

```
cmd/boi/          Entry point
internal/
  cli/            Cobra command definitions
  agent/          ReAct loop (plan → execute → review)
  persona/        Persona registry and loader
  skill/          Skill system (markdown-based plugins)
  memory/         Phantom DB with weight engine
  tui/            Bubbletea terminal UI
  config/         Configuration management
  workspace/      Workspace detection
```

## Pull Request Process

1. Fork the repo and create a branch from `main`.
2. Make your changes — keep them focused on one concern.
3. Ensure `go fmt ./...` and `go vet ./...` pass.
4. Update README.md or documentation if your change affects the user-facing API.
5. Open a PR against `main` using the [PR template](.github/PULL_REQUEST_TEMPLATE.md).
6. A maintainer will review and merge.

## Community Guidelines

- Be respectful and constructive. Read our [Code of Conduct](CODE_OF_CONDUCT.md).
- Focus on the problem, not the person.
- Help others learn — explain your reasoning in PR descriptions.
- Keep discussions in public issues and PRs.

## Questions?

Open a [GitHub Discussion](https://github.com/wersoul-source/BOI-CLI/discussions) or
start an issue with the "question" label.

Built by **Kampun (คำปัน)** and the **BOI Family**.
