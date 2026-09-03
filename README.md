# BOI CLI

BOI CLI is a bounded, tool-using Agent runtime for terminal workspaces. It has
one Core Persona, `boi`, while each user gives their Agent instance its own name
on the first TUI launch.

The product is organized into six Blocks: Service, Core, Various Equipment,
Runtime, Agent Folder, and SubAgent. Work 1 connects the first five into one
controlled TUI/CLI execution path; SubAgent execution remains disabled.

## Work 1 capabilities

- One shared Agent Service and Engine for TUI and `boi ask`.
- Observe → Decide → Authorize → Act → Verify → Recover lifecycle with bounded
  steps, tools, tokens, time, and recovery.
- Explicit Provider qualification before a Provider can enter Agent routing.
- Fail-closed Local Registry with at most 15 active Skills and 15 active Tools.
- Capability Broker for risk classification, approval, timeout, and execution.
- Workspace Sandbox path boundary plus an interactive TUI approval panel.
- One `agent-folder` tray: diagnostics in `bin`, deliverables and manifests in
  `output`.
- Schema-v1 JSON output, stable exits, stdin/argv input, and deterministic
  read-only Automation.

BOI CLI does not fall back to a simulated AI response. A configured Provider
must pass `boi provider qualify` before Agent tasks can use it.

## Requirements

- Go 1.24.2 or a compatible later toolchain
- A supported Provider configured through environment or setup
- A terminal capable of running the TUI

## Build and start

```text
go build -o boi ./cmd/boi
boi init
boi setup
boi provider qualify <provider-name>
boi registry init
boi
```

`boi registry init` is safe and non-overwriting. Normal runtime initialization
also creates missing indexes so migrated workspaces continue without exposing
loose, unindexed capability files.

The first TUI start asks for the Agent instance name and stores it in
`.boi/agent.yaml`. This name is not a Persona or a Provider identity.

## Non-interactive use

```text
boi ask explain this repository
Get-Content task.txt | boi ask --json --idempotency-key task-001
```

Work 1 Automation is read-only. A Tool call requiring approval is denied in
non-interactive mode; the process never waits for an approval prompt. JSON goes
to stdout as one object and verbose diagnostics go to stderr. See the
[Automation contract](docs/operations/AUTOMATION_CONTRACT.md).

## Command groups

| Command | Purpose |
|---|---|
| `boi` | Start the TUI |
| `boi ask` | Run the bounded Agent non-interactively |
| `boi setup` | Configure Providers interactively |
| `boi provider list/switch/qualify` | Manage and qualify Provider candidates |
| `boi registry init/list/add` | Manage explicit Skill and Tool indexes |
| `boi config` / `boi model` | Inspect or change runtime configuration |
| `boi doctor` | Run local health checks |
| `boi skill` / `boi memory` | Manage installed Skills and local memory |
| `boi persona` | Show the fixed Core Persona compatibility contract |
| `boi version` / `boi upgrade` | Inspect or update the binary |

Use `boi <command> --help` for the live flag contract. The legacy `boi run`
shell helper is not part of the Agent Tool authority path.

## Workspace layout

```text
workspace/
├── .boi/
│   ├── agent.yaml
│   ├── config.yaml
│   ├── provider-profiles/
│   ├── registry/
│   │   ├── skills.json
│   │   └── tools.json
│   ├── skills/
│   └── memory/
└── agent-folder/
    ├── bin/
    └── output/
```

Completed task manifests live under `agent-folder/output/<task-id>/`. Failed or
cancelled task diagnostics remain under `agent-folder/bin/<task-id>/`. Cleanup
is bin-only and dry-run by default.

## Safety and current limits

- The Workspace Sandbox enforces path boundaries; it is not OS or container
  isolation.
- Mutating Tools require exact interactive approval. Side-effecting Automation
  remains disabled.
- MCP primitives exist, but full discovery and Library routing are not yet on
  the main Agent path.
- SubAgent execution is disabled until its separate authority and evaluation
  gate is accepted.
- BOI CLI is network-capable and is not designed as offline-first.
- Android ARM64 cross-build is verified; physical S25+ runtime acceptance is a
  separate, still-required validation path.

## Architecture and release status

- [Work 1 plan](docs/planning/WORK_1_PLAN.md)
- [CLI command reference](docs/reference/CLI_COMMANDS.md)
- [Work 1 release and rollback notes](docs/operations/WORK_1_RELEASE.md)
- [Project handoff](HANDOFF.md)

License: MIT
