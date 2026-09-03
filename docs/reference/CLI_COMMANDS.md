# BOI CLI Command Reference

This reference describes the Work 1 command contract. The executable remains
authoritative; use `boi <command> --help` to inspect the exact flags in the
installed version.

## `boi` — TUI

Running `boi` without a subcommand starts the terminal UI. On first use, BOI
asks for an Agent instance name and stores it in `.boi/agent.yaml`. Runtime
Persona remains fixed to the Core Persona `boi`.

Important TUI controls:

| Input | Action |
|---|---|
| `Enter` | Submit input |
| `Tab` | Complete a slash command |
| `Esc` or `Ctrl+C` | Cancel an active Agent task; otherwise quit |
| `Ctrl+Q` | Quit the TUI |
| `Ctrl+L` | Clear chat |
| `/help` | Show TUI commands |
| `/persona` | Show fixed Core Persona and Agent name |
| `/providers` | Show qualified Providers |

Mutating Tool calls display an Approval Panel. The decision applies to the
exact request shown; a Skill or Model response cannot grant itself authority.

## `boi ask [query...]` — non-interactive Agent

```text
boi ask explain this repository
Get-Content task.txt | boi ask --json --idempotency-key inspect-001
```

Input is read from argv when any query arguments are present. Otherwise BOI
accepts bounded UTF-8 stdin when stdin is piped. Missing input returns an error
without prompting.

Flags:

| Flag | Meaning |
|---|---|
| `--steps <n>` | Maximum Agent steps; default 15 |
| `--verbose` | Write diagnostics to stderr |
| `--json` | Write exactly one schema-v1 result object to stdout |
| `--idempotency-key <key>` | Host-hashed Automation key; requires `--json` |

Work 1 non-interactive execution is read-only. Approval-required mutation is
reported as denied and is never executed or left waiting for input. Stable
process exits are:

| Exit | Class |
|---:|---|
| 0 | completed |
| 1 | internal error |
| 2 | invalid input |
| 3 | denied |
| 4 | cancelled |
| 5 | unavailable |
| 6 | verification failed |

The complete JSON fields, stream rules, idempotency boundary, and examples are
defined in [AUTOMATION_CONTRACT.md](../operations/AUTOMATION_CONTRACT.md).

## Workspace and setup

### `boi init`

Initializes the BOI workspace structure without intentionally replacing user
task output.

### `boi setup`

Runs the interactive Provider configuration wizard. Configuration alone does
not qualify a Provider for Agent routing.

### `boi doctor`

Checks the local Go version, workspace, configuration, Persona compatibility,
Skills, memory, Providers, binary, and OS state. A health check is not proof of
a successful live Provider call.

## Providers

### `boi provider list`

Lists configured Provider candidates and their state.

### `boi provider switch <name>`

Selects the configured default Provider.

### `boi provider qualify <name>`

Runs the versioned BOI behavioral probe suite and stores a credential-free
capability profile in `.boi/provider-profiles/`. Only Providers with a passing
completion profile can enter the Agent Router. Capability composition is
fail-closed across qualified failover candidates.

Provider qualification can make real network or billable API calls; it runs
only when explicitly requested.

### `boi model`

Changes the default model for the current Provider. Changing Provider or model
does not change Agent identity or Core Persona.

## Capability registry

### `boi registry init`

Creates missing `.boi/registry/skills.json` and `tools.json` without
overwriting existing indexes.

### `boi registry list`

Lists explicitly registered capability entries and their state. Loose files
are deliberately excluded.

### `boi registry add`

Adds one installed Skill or Tool to its explicit Local Registry. Inspect
`boi registry add --help` for the required kind, name, source, and capability
metadata in the current binary.

At runtime the selected active set is capped at 15 Skills and 15 Tools.
Installed does not mean eligible, selected, active, or authorized.

## Core Persona compatibility

Work 1 has one Persona: `boi`. The separately stored Agent instance name is the
user-facing identity.

| Command | Behavior |
|---|---|
| `boi persona list` | Shows only the fixed Core Persona `boi` |
| `boi persona switch boi` | Normalizes the compatibility config to `boi` |
| `boi persona switch <other>` | Returns an explicit retired-switching error |
| `boi persona init` | Installs `boi.yaml` only when missing |
| `boi persona wizard` | Explains that Persona selection is retired |

Legacy Persona files are preserved during migration but are not selectable
runtime identities.

## Skills and memory

`boi skill init/list/show` manages installed Skill material. Runtime exposure
still requires the explicit registry and task selection; Skill instructions do
not grant Tool authority.

`boi memory init/search/save/stats/repomap` manages local cross-session memory
and repository map data.

## Configuration and maintenance

- `boi config` shows or edits workspace configuration.
- `boi version` reports the binary version.
- `boi upgrade` updates through the project's upgrade mechanism.
- `boi completion` generates shell completion scripts.
- `boi uninstall` removes the installed CLI according to its command prompts.
- `boi weight` explains Weight Engine scores.

The legacy `boi run` command executes a shell command directly. It is not part
of the Agent Engine, Capability Broker, approval, or Tool authority contract;
do not use it as evidence that an Agent Tool action was authorized.

## Removed claims

- There is no six-Persona runtime team.
- There is no selectable Persona in the main Agent path.
- There is no simulated Agent response when Provider credentials or
  qualification are missing.
- Android cross-build success is not physical S25+ runtime acceptance.
