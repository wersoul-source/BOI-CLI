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

## Work 1 status

Work 1 is complete for the single-Agent host path. The Core Persona, qualified
Provider Router, bounded Registry, Broker, Sandbox, Agent Folder, TUI approval,
and read-only Automation all use one Agent Service contract. SubAgent and
side-effecting Automation remain deliberately disabled for Work 2.

Built-binary acceptance runs on Windows and Linux. The shared simulation covers
Unicode and spaced paths, nested and large folders, approval denial, traversal,
binary and missing inputs, corrupt registries, and unqualified Providers. The
Linux suite additionally verifies symlink escape. Linux ARM64 and Android ARM64
cross-builds are release gates; this establishes the owner-approved Termux/S25+
compatibility baseline without claiming that a physical handset was exercised.

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

`boi setup` preserves unrelated `.env` values, replaces only its managed
Provider section, keeps a timestamped backup, uses private file permissions on
Unix-like systems, and adds local Git excludes for BOI secrets. Setup does not
qualify a Provider; qualification remains an explicit behavioral test.

The first TUI start asks for the Agent instance name and stores it in
`.boi/agent.yaml`. This name is not a Persona or a Provider identity.

## Work 1 quickstart — create your first artifact

Work inside the project directory that BOI is allowed to inspect and change.
The directory becomes the Workspace Sandbox boundary, so start BOI from the
repository or folder that contains the work you want the Agent to perform.

### 1. Build BOI

Windows PowerShell:

```powershell
go build -o boi.exe ./cmd/boi
```

Linux, WSL, or Termux:

```bash
go build -o boi ./cmd/boi
chmod +x ./boi
```

The examples below use `boi`. Use `.\boi.exe` on Windows when the executable
is not installed on `PATH`, or `./boi` on Linux/Termux.

### 2. Initialize the Workspace

```text
boi init
boi registry init
```

Initialization creates the `.boi` runtime state and the bounded Local Registry.
It does not delete existing project files or Agent output.

### 3. Connect and qualify a Provider

```text
boi setup
boi provider list
boi provider qualify <provider-name>
boi doctor
```

`boi setup` opens the Provider wizard. Select a Provider and model, then enter
the API credential when prompted. Use the configured name shown by
`boi provider list` in the qualification command; for example:

```text
boi provider qualify openai
```

Qualification sends a real behavioral probe suite to the Provider and may
consume billable API tokens. Configuration alone is not enough: an unqualified
Provider is excluded from the Agent Router.

### 4. Start the TUI

```text
boi
```

On the first launch, give the Agent instance a name. Press `Enter` on the
Splash Screen to open the Chat. The Core Persona remains `boi` regardless of
the instance name.

### 5. Ask BOI to create a file

Use this safe first task inside an initialized test Workspace:

```text
Create a file named hello-boi.md in this workspace. Add a title, a short
description of this repository, and a checklist of three useful next steps.
Read the available project context before writing, and report the final path.
```

Expected flow:

1. BOI inspects the Workspace with read-only Tools.
2. The Model proposes a `workspace.write` Tool Call.
3. TUI replaces the input box with an Approval Panel showing purpose, target,
   risk, and preview.
4. Press `A` to approve that exact write once, `R` to reject it, or `Esc` to
   cancel the task. `Enter` never approves a write.
5. BOI verifies the Tool Result and reports the Task ID and manifest path.

Inspect the result without leaving the TUI:

```text
/ls
/read hello-boi.md
```

The created file remains in the Workspace. BOI records completed task evidence
under:

```text
agent-folder/output/<task-id>/manifest.json
```

Failed, rejected, cancelled, or recovery diagnostics remain under
`agent-folder/bin/<task-id>/` and are not presented as completed deliverables.

### 6. Try a repository task

After the first file succeeds, try a bounded task with an explicit output:

```text
Inspect this repository and create WORKSPACE_REVIEW.md. Summarize the project
structure, identify three concrete risks supported by files you inspected, and
propose a five-step improvement plan. Do not modify any other file.
```

Review the Approval Panel carefully before authorizing the write. BOI's
Workspace Sandbox limits filesystem paths, but it is not OS/container
isolation. Run unfamiliar projects inside an appropriately isolated operating
environment when stronger protection is required.

### TUI controls used in Work 1

| Input | Action |
|---|---|
| `Enter` | Send a Chat message; never approves a Tool Call |
| `Ctrl+N` | Insert a newline in the input box |
| `Tab` | Complete a slash command |
| `Esc` or `Ctrl+C` | Cancel an active task; quit when idle |
| `Ctrl+Q` | Quit immediately |
| `Ctrl+L` | Clear the visible Chat |
| `/workspace` | Show the active Sandbox root |
| `/ls [path]` | List a directory inside the Workspace |
| `/read <path>` | Read a text file inside the Workspace |
| `/providers` | Show qualified Provider state |
| `/persona` | Show the fixed Core Persona and Agent instance name |

### Troubleshooting the first run

| Symptom | Meaning and action |
|---|---|
| `no qualified providers` | Run `boi provider list`, then `boi provider qualify <name>`. |
| Provider qualification fails | Check API key, Base URL, model name, network access, and Provider quota. |
| A write is denied by `boi ask` | Work 1 non-interactive Automation is read-only; use the TUI approval flow. |
| `capability registry` error | Run `boi registry init`; inspect rather than overwrite an existing invalid Registry. |
| Workspace path is rejected | Keep the target inside the Workspace root and avoid symlink/traversal paths. |
| Binary file is rejected | Work 1 Workspace reading accepts bounded text files, not binary content. |

Never commit `.env`, API credentials, or Provider backups. BOI adds local Git
exclusions during setup, but the user remains responsible for repository and
credential hygiene.

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
Informational commands such as `--help` and `version` resolve the workspace
without creating `.boi` or `agent-folder` state.

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
  recommended device check rather than a Work 1 host-release blocker. Linux
  runtime parity is the accepted Termux/S25+ simulation baseline.
- `boi upgrade` downloads only from the canonical release repository and
  verifies the published SHA-256 checksum before replacing the binary.

## Verification

```text
go test -count=1 ./...
go vet ./...
go build ./...
```

CI runs these gates on Windows and Linux and cross-builds Android ARM64. Manual
WSL parity can be exercised with
`scripts/acceptance/linux_folder_simulation.py` and a Linux BOI binary. The
simulation uses a local OpenAI-compatible fixture and does not prove a live,
third-party Provider account.

## Architecture and release status

- [Work 1 plan](docs/planning/WORK_1_PLAN.md)
- [CLI command reference](docs/reference/CLI_COMMANDS.md)
- [Work 1 release and rollback notes](docs/operations/WORK_1_RELEASE.md)
- [Project handoff](HANDOFF.md)

License: MIT
