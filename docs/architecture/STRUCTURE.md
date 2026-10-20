# BOI CLI Repository Structure

Package ownership and dependency rules now live in
[BLOCK_ARCHITECTURE.md](BLOCK_ARCHITECTURE.md). The rules are enforced by
`internal/architecture/deps_test.go`.

## Safety boundaries

The TUI and `boi ask` build their Agent through `app.BuildAgent` and run it
through `internal/runtime/agent.Service`. Model tool proposals enter the
host-owned Broker. Workspace reads are automatic; writes, processes, and MCP
tools require an exact, expiring approval displayed by the TUI.
Non-interactive CLI use never approves these actions automatically.

The workspace boundary (`internal/runtime/workspace`) validates lexical and
canonical paths, including symlink targets. It constrains filesystem paths
only and must not be described as process, container, or operating-system
isolation.

The command deny-list in `internal/equipment/tools/process/sandbox.go` is a
best-effort guard against obviously destructive text, not isolation.
Obfuscated commands (variable expansion, encoded payloads, nested
interpreters) evade it. The real controls are the Broker approval step and
the workspace path boundary.

## Runtime data boundary

The repository does not own the developer's `.boi` runtime directory. A
sanitized fixture lives at `tests/testdata/workspace/.boi`. Real provider keys,
memory entries, backup files, and machine-local configuration must remain
ignored.
