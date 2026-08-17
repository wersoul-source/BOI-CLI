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

1. Make the TUI usable against an explicit Workspace Sandbox.
2. Provide bounded, read-only `/workspace`, `/ls`, and `/read` operations.
3. Route TUI chat through one Agent Service with persona and memory context.
4. Add visible busy, error, timeout, and cancellation states.
5. Define typed tool calls, tool results, approvals, usage, and stop reasons.
6. Implement the observe, decide, authorize, act, verify, and recover loop.
7. Add provider error classification and bounded retry/failover.
8. Add write and process tools only after the TUI has a visible approval flow.
9. Route non-interactive CLI commands through the same Agent Service.
10. Add MCP only after the local tool contract and recovery tests pass.
11. Add task-level and adversarial evaluation before enabling subagents.

The Workspace Sandbox is a canonical filesystem-path boundary. It prevents
path traversal and symlink escape, but it is not an operating-system sandbox
for spawned processes.

## Closing gate: CLI product contract

Command names, inputs, streams, exit codes, prompts, destructive-action rules,
and machine-readable output will be finalized after the owner supplies the
intended function of each BOI CLI command.
