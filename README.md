# BOI Agent Suite

[English](README.md) | [ภาษาไทย](README-TH.md)

BOI is a bounded, tool-using Agent for terminal workspaces. It reads your
project, proposes actions, asks before changing anything, verifies what it
did, and leaves a manifest of the result.

v1.5 rebuilt the codebase around one idea: **a fixed spine with pluggable
Blocks**. The Agent's core stays the same, and new capabilities plug in
without editing it. Since then the Agent talks to each model natively where
the model proves it can, reads and searches files in fewer turns, and is
composed per workspace from Block packs (built-in Tools, MCP servers, an
optional SubAgent) that `boi env` reports from what is really loaded.

Release binaries are built for Linux and Windows (amd64 and arm64). macOS and
Android compile but are not release targets.

## Quick start

**1. Install.** Nothing else is needed. Open a new terminal afterwards so `boi` is on your PATH.

Linux:

```bash
curl -fsSL https://raw.githubusercontent.com/wersoul-source/BOI-CLI/master/scripts/install.sh | bash
```

Windows (PowerShell):

```powershell
irm https://raw.githubusercontent.com/wersoul-source/BOI-CLI/master/scripts/install.ps1 | iex
```

No release for your system yet? [Build from source](#build-from-source).

**2. Get ready.** In the folder of the project you want the Agent to work on:

```text
boi quickstart
```

It finds a model for you and checks what that model can do:

- a model running on your computer (Ollama, LM Studio, llama.cpp) is detected
  and offered by name; nothing to type, no API key
- or choose a hosted provider (OpenAI, Anthropic, Cloudflare and others) and
  enter its API key
- the settings go into `.env` in the project, kept out of Git

No model yet? The quickest free start is [Ollama](https://ollama.com): install
it, run `ollama pull qwen2.5-coder:7b`, then `boi quickstart`.

**3. Use it.**

```text
boi                                 chat and work on this project
boi ask "what does main.go do?"     one read-only answer
```

Every edit and every command asks you first and shows what it will do.
Something wrong? `boi doctor` checks the setup and `boi env` shows what the
Agent can use right now. More in [GET_STARTED.md](GET_STARTED.md).

## Architecture

```text
            ┌──────────────── fixed spine ────────────────┐
            │  Core      identity, qualification, Persona │
            │  Runtime   engine, Broker, approval, LLM    │
            └──────────────────────┬──────────────────────┘
                                   │ ports (block/port, llm.Provider, ...)
      ┌──────────────┬─────────────┼──────────────┬──────────────┐
  Equipment       Service      Agent Folder     SubAgent       (yours)
  tools, skills,  providers,   bin / output     agent.delegate,
  memory, MCP     config       manifests        optional
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

### Choose the Blocks of an Agent

`.boi/environment.json` lists the Block packs a workspace's Agent uses and
whether each is required. Without it, every built-in pack is used. A
required pack that is missing or incompatible stops start-up with the reason.

```json
{"schema_version": 1, "name": "no-shell", "blocks": [{"id": "workspace-tools", "required": true}]}
```

Ready-made environments: `boi env presets`, then `boi env use minimal`
(read-only) or `boi env use coding` (read, write, run commands with
approval). `boi env` shows what was composed.

Built-in packs: `workspace-tools` (list, read, search, write),
`workspace-read` (list, read, search), `process-tools` (`process.run`) and
`subagent` (`agent.delegate`: hands a sub-task to a helper limited to the
Tools you grant; see [ADR-003](docs/architecture/ADR-003-subagent.md)).

An entry with `mcp` plugs in a local MCP server's tools, approval-gated,
without building BOI:

```json
{"id": "github", "required": false, "mcp": {"command": "github-mcp-server", "args": ["stdio"]}}
```

MCP entries are commands that run on your machine, so a workspace's MCP
servers start only after you review `environment.json` and run
`boi env trust`. Editing the file withdraws the trust. See
[ADR-002](docs/architecture/ADR-002-block-packs.md).

An entry with `skills` plugs in a folder of Skill files (`*.skill.md`,
relative to `.boi`). Skills are text: they add no Tool and need no trust, and
the model decides whether to load one. A Skill works with or without the
Tools it names. See [ADR-004](docs/architecture/ADR-004-skill-packs.md).

```json
{"id": "team-skills", "required": false, "skills": {"dir": "skills/team"}}
```

### Slow or local models

A server on `localhost` (Ollama, llama.cpp, LM Studio) needs no API key: set
`PSC_1_NAME=ollama`, `PSC_1_BASE_URL` and `PSC_1_MODEL`. The task and request
limits are made for hosted APIs (2 minutes, 64,000 tokens) and can be raised
with `BOI_TASK_TIMEOUT`, `BOI_REQUEST_TIMEOUT`, `BOI_MAX_TOKENS`,
`BOI_MAX_TOOL_CALLS` and `BOI_MAX_STEPS`; a local server gets 3 minutes per
qualification probe because its first answer loads the model. See
[GET_STARTED.md](GET_STARTED.md).

### Two ways to plug in

| Path | When | How |
|---|---|---|
| Compile time | A Tool written in Go | Implement `port.Tool`, add it to a pack in `app.BuiltinPacks` and to `app.DefaultCapabilityIndexes`. The Broker is not edited. |
| Run time | Anything outside the binary | Expose it as an MCP server. `equipment/tools/mcp` adapts each MCP tool into an approval-gated Tool. |

## How a task runs

```text
Observe → Decide → Authorize → Act → Verify → Recover
```

1. Core selects at most 15 Tools and 15 Skills for the task.
2. The Model proposes up to 5 Tool calls per turn, as native tool calls when
   its Provider qualified for them, otherwise as a `<boi-action>` envelope.
   It cannot set risk or approval; the Broker assigns them from the Tool's
   spec. A batch of automatic reads runs concurrently; anything else runs in
   order.
3. Reads run automatically. Writes, processes and MCP calls need your exact
   approval in the TUI. Non-interactive mode denies them.
4. Large or old tool outputs are shortened in the model context but kept in
   full in a per-task store; the model reads any part back exactly with
   `context.expand(ref, lines)`.
5. The Tool verifies its own effect (for example, it reads a written file
   back) before the step counts as done.
6. Completed work lands in `agent-folder/output/<task-id>/` with a manifest.
   Failed or cancelled work stays in `agent-folder/bin/<task-id>/`.

## Build from source

Requires Go 1.24.2 or later.

```bash
git clone https://github.com/wersoul-source/BOI-CLI.git
cd BOI-CLI
go build -trimpath -o boi ./cmd/boi      # Windows: go build -o boi.exe ./cmd/boi
sudo install boi /usr/local/bin/         # or keep ./boi
```

Then run `boi quickstart` in your project. The individual steps it performs,
for when you want them one by one:

```bash
boi init                        # create .boi state (non-destructive)
boi setup                       # choose a Provider (wizard)
boi provider qualify <name>     # real behavioral test; uses API tokens for hosted models
boi doctor                      # health check
boi                             # start the TUI
```

On first launch the TUI asks for your Agent's name. The Core Persona is always
`boi`; the name belongs to your Agent instance only.

A Provider that has not passed `boi provider qualify` never enters the Agent
Router. BOI does not fall back to simulated answers. Qualification also
decides whether native tool calling is used (`native_tool_schema`).
Profiles from v1.5 do not load in v1.6 or later, and every profile expires after 30 days; qualify again when `boi env` reports none.

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
| `boi quickstart` | Find a model, save it, check it: the guided first run |
| `boi init` / `boi setup` | Initialize the workspace / configure Providers |
| `boi provider list\|switch\|qualify` | Manage and qualify Providers |
| `boi registry init\|list\|add` | Manage the explicit Tool and Skill index |
| `boi doctor` | Local health checks |
| `boi env` | Which Blocks the Agent is composed of: active, loaded or declared, and why |
| `boi env presets\|use\|trust` | List and apply presets; trust a workspace environment's MCP servers |
| `boi skill` / `boi memory` | Manage Skills and local memory |
| `boi config` / `boi model` | Inspect or change configuration |
| `boi persona` / `boi weight` | Show the Core Persona / memory weights |
| `boi run` | Run one approved shell command in the workspace |
| `boi uninstall` | Remove BOI from this machine |
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

### Use one Provider in every project

Put shared Provider settings in `~/.config/boi/.env` (same `PSC_*` keys as a
workspace `.env`). A workspace `.env` still wins when both set a key, and
shell variables win over both. Qualification results are cached in
`~/.config/boi/provider-profiles`, so a Provider qualified once is ready in
every new project. `.env` values may be quoted and may start with `export`.

Any OpenAI-compatible endpoint works by setting `PSC_1_NAME=openai` and
`PSC_1_BASE_URL`. Verified so far: Cloudflare Workers AI
(`scripts/acceptance/cloudflare_smoke.sh`).

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

To measure task success, provider calls, tokens and time on a fixed task set,
and compare against an earlier version:

```bash
BASE_REF=<commit-or-tag> bash scripts/acceptance/benchmark.sh
```

## Safety and limits

- The workspace sandbox enforces path boundaries, including symlinks. It is
  not OS or container isolation; run unfamiliar projects in an isolated VM.
- The command deny-list is a best-effort guard, not isolation. The real
  controls are the Broker approval step and the path boundary.
- The SubAgent is optional and not in any preset; its benchmark gate is still
  open.
- MCP servers run as local processes (stdio only) and start with the
  environment minus `PSC_*` variables; they are not sandboxed beyond that.
- BOI needs the network to reach Providers; it is not offline-first.
- Only Linux has been exercised end to end. Other platforms are compile
  checks.

## Contributing

Read [BLOCK_ARCHITECTURE.md](docs/architecture/BLOCK_ARCHITECTURE.md) before
adding a package, and [CONTRIBUTING.md](CONTRIBUTING.md) for workflow.
Project history and handoff notes: [HANDOFF.md](HANDOFF.md).

License: MIT
