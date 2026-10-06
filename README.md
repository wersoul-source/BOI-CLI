# BOI Agent Suite v1.5

[English](README.md) | [ภาษาไทย](README-TH.md)

BOI is a bounded, tool-using Agent for terminal workspaces. It reads your
project, proposes actions, asks before changing anything, verifies what it
did, and leaves a manifest of the result.

v1.5 rebuilds the codebase around one idea: **a fixed spine with pluggable
Blocks**. The Agent's core stays the same, and new capabilities plug in
without editing it.

Linux is the first supported platform. Windows, macOS and Android compile but
are not yet release targets.

## Architecture

```text
            ┌──────────────── fixed spine ────────────────┐
            │  Core      identity, qualification, Persona │
            │  Runtime   engine, Broker, approval, LLM    │
            └──────────────────────┬──────────────────────┘
                                   │ ports (block/port, llm.Provider, ...)
      ┌──────────────┬─────────────┼──────────────┬──────────────┐
  Equipment       Service      Agent Folder     SubAgent       (yours)
  tools, skills,  providers,   bin / output     gated,
  memory, MCP     config       manifests        disabled
```

- **Spine** (`internal/core`, `internal/runtime`): fixed. It defines the
  contracts every Block must satisfy and never imports a Block.
- **Blocks** (`internal/equipment`, `internal/service`, `internal/agentfolder`,
  `internal/subagent`): pluggable. A Block depends only on the spine and never
  on another Block.
- **Composition root** (`internal/app`): the only place that wires Blocks
  into the spine. TUI and CLI both build their Agent through
  `app.BuildAgent`.

The rules are enforced by `internal/architecture/deps_test.go`. A change that
breaks them fails `go test ./...`. Full layout:
[docs/architecture/BLOCK_ARCHITECTURE.md](docs/architecture/BLOCK_ARCHITECTURE.md).

### Two ways to plug in

| Path | When | How |
|---|---|---|
| Compile time | A Tool written in Go | Implement `port.Tool`, add it to `app.BuiltinTools` and to the capability index. The Broker is not edited. |
| Run time | Anything outside the binary | Expose it as an MCP server. `equipment/tools/mcp` adapts each MCP tool into an approval-gated Tool. |

## How a task runs

```text
Observe → Decide → Authorize → Act → Verify → Recover
```

1. Core selects at most 15 Tools and 15 Skills for the task.
2. The Model proposes one Tool call. It cannot set risk or approval; the
   Broker assigns them from the Tool's spec.
3. Reads run automatically. Writes, processes and MCP calls need your exact
   approval in the TUI. Non-interactive mode denies them.
4. The Tool verifies its own effect (for example, it reads a written file
   back) before the step counts as done.
5. Completed work lands in `agent-folder/output/<task-id>/` with a manifest.
   Failed or cancelled work stays in `agent-folder/bin/<task-id>/`.

## Quick start (Linux)

Requires Go 1.24.2 or later and an API key for a supported Provider.

```bash
git clone https://github.com/wersoul-source/BOI-CLI.git
cd BOI-CLI
go build -trimpath -o boi ./cmd/boi
sudo install boi /usr/local/bin/      # or keep ./boi
```

In the project you want the Agent to work on:

```bash
boi init                        # create .boi state (non-destructive)
boi registry init               # create the bounded Tool/Skill index
boi setup                       # choose a Provider and enter the API key
boi provider qualify <name>     # real behavioral test; uses API tokens
boi doctor                      # health check
boi                             # start the TUI
```

On first launch the TUI asks for your Agent's name. The Core Persona is always
`boi`; the name belongs to your Agent instance only.

A Provider that has not passed `boi provider qualify` never enters the Agent
Router. BOI does not fall back to simulated answers.

### First task

```text
Create hello-boi.md with a title, a short description of this repository,
and three useful next steps. Read the project first and report the path.
```

When the write is proposed, the input box becomes an Approval Panel. Press `A`
to approve that exact write once, `R` to reject, `Esc` to cancel. `Enter`
never approves.

## Non-interactive use

```bash
boi ask "explain this repository"
cat task.txt | boi ask --json --idempotency-key task-001
```

`--json` writes one versioned object to stdout and diagnostics to stderr.
Automation is read-only: any call that needs approval is denied, never
awaited. Exit codes: `0` completed, `1` internal, `2` invalid input,
`3` denied, `4` cancelled, `5` unavailable, `6` verification failed. See the
[Automation contract](docs/operations/AUTOMATION_CONTRACT.md).

## Commands

| Command | Purpose |
|---|---|
| `boi` | Start the TUI |
| `boi ask` | Run the Agent non-interactively |
| `boi init` / `boi setup` | Initialize the workspace / configure Providers |
| `boi provider list\|switch\|qualify` | Manage and qualify Providers |
| `boi registry init\|list\|add` | Manage the explicit Tool and Skill index |
| `boi doctor` | Local health checks |
| `boi skill` / `boi memory` | Manage Skills and local memory |
| `boi config` / `boi model` | Inspect or change configuration |
| `boi version` / `boi upgrade` | Show version / checksum-verified upgrade |

### TUI keys

| Key | Action |
|---|---|
| `Enter` | Send; never approves a Tool call |
| `Ctrl+N` | New line |
| `Tab` | Complete a slash command |
| `Esc` / `Ctrl+C` | Cancel the active task; quit when idle |
| `Ctrl+Q` | Quit |
| `/ls [path]`, `/read <path>` | Inspect the workspace |
| `/workspace`, `/providers`, `/persona` | Show sandbox root, Provider state, identity |

## Workspace layout

```text
your-project/
├── .boi/
│   ├── agent.yaml             Agent instance name
│   ├── config.yaml
│   ├── provider-profiles/     qualification results
│   ├── registry/              tools.json, skills.json (15/15 active max)
│   ├── skills/
│   └── memory/
└── agent-folder/
    ├── bin/                   drafts, logs, failed and cancelled tasks
    └── output/                deliverables and manifests
```

Never commit `.env` or API keys. `boi setup` adds local Git excludes, keeps a
timestamped backup, and writes the file with private permissions.

## Verify on Linux

```bash
make smoke
```

runs vet, race tests, the architecture rules, and a built-binary simulation
of nine workspace scenarios against a local fake Provider. To add a real
Provider round trip:

```bash
PSC_1_NAME=openai PSC_1_API_KEY=... PSC_1_MODEL=... make smoke
```

CI runs the same Linux gates plus staticcheck and a coverage floor, and
compile-checks linux/arm64, windows, darwin and android.

## Safety and limits

- The workspace sandbox enforces path boundaries, including symlinks. It is
  not OS or container isolation; run unfamiliar projects in an isolated VM.
- The command deny-list is a best-effort guard, not isolation. The real
  controls are the Broker approval step and the path boundary.
- SubAgent execution is disabled until its evaluation gate is accepted.
- MCP Tools can be registered, but automatic MCP discovery is not yet on the
  main path.
- BOI needs the network to reach Providers; it is not offline-first.
- Only Linux has been exercised end to end. Other platforms are compile
  checks.

## Contributing

Read [BLOCK_ARCHITECTURE.md](docs/architecture/BLOCK_ARCHITECTURE.md) before
adding a package, and [CONTRIBUTING.md](CONTRIBUTING.md) for workflow.
Project history and handoff notes: [HANDOFF.md](HANDOFF.md).

License: MIT
