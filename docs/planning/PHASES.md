# BOI CLI Delivery Phases

## Part 1: Repository structure

- Add characterization tests for public CLI defaults and workspace/config
  behavior.
- Introduce `internal/app` as the process runtime context.
- Separate CLI and TUI into transport packages.
- Group provider, tools, memory, configuration, and platform integrations by
  ownership.
- Remove machine-local `.boi` data from version control and keep sanitized
  fixtures under `tests/testdata`.
- Reorganize documentation, packaging, asset generation, and release scripts.
- Pass formatting, tests, vet, build, and CLI smoke checks.

## Part 2: TUI-first Agent runtime

- [x] Make the TUI usable against an explicit Workspace Sandbox.
- [x] Provide bounded, read-only `/workspace`, `/ls`, and `/read` operations.
- [x] Route TUI chat through one Agent Service with persona and memory context.
- [x] Add visible busy, error, timeout, and cancellation states.
- [x] Define typed tool calls, tool results, approvals, usage, and stop reasons.
- [x] Implement the bounded observe, decide, authorize, act, verify, and recover
  kernel loop. Capability ports remain disconnected until Task 8.
- [x] Add provider error classification and bounded retry/failover.
- [x] Add write and process tools only after the TUI approval flow is connected
  to the Agent loop and capability broker.
- [x] Route non-interactive CLI commands through the same Agent Service.
- [x] Add MCP only after the local tool contract and recovery tests pass.
- [x] Add task-level and adversarial evaluation before enabling subagents.

The TUI Approval Panel is connected to the Agent runtime. Host policy assigns
risk and approval classes; model-driven write, process, and registered MCP
capabilities require an exact, expiring approval. Subagents remain disabled.

The Workspace Sandbox is a canonical filesystem-path boundary. It prevents
path traversal and symlink escape, but it is not an operating-system sandbox
for spawned processes.

## Closing gate: CLI product contract

Command names, inputs, streams, exit codes, prompts, destructive-action rules,
and machine-readable output will be finalized after the owner supplies the
intended function of each BOI CLI command.
