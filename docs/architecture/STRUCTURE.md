# BOI CLI Repository Structure

This document is the source of truth for package ownership after the Phase 1
reorganization.

## Dependency direction

```text
cmd/boi
  -> internal/app
  -> internal/transport/{cli,tui}
       -> internal/agent
       -> internal/provider
       -> internal/memory
       -> internal/tool
       -> internal/config
       -> internal/persona
       -> internal/workspace
       -> internal/platform
```

Transport packages parse input and render output. They must not own agent,
provider, memory, or tool policy. `internal/app` owns process-wide runtime
context and will become the composition root for shared services in Phase 2.

## Package ownership

| Path | Responsibility |
|---|---|
| `cmd/boi` | Minimal executable entry point |
| `internal/app` | Runtime context, lifecycle, and service composition |
| `internal/transport/cli` | Cobra commands and CLI rendering |
| `internal/transport/tui` | Bubble Tea presentation |
| `internal/agent` | Agent state and execution loop |
| `internal/provider` | Provider contract, routing, adapters, and catalog |
| `internal/tool` | Bounded filesystem, process, and MCP capabilities |
| `internal/memory` | Persistent memory and weight policy |
| `internal/config` | Configuration and environment loading |
| `internal/platform` | Terminal, logging, update, and OS integration |
| `internal/persona` | Persona definitions and loading |
| `internal/skill` | Skill definitions and loading |
| `internal/workspace` | Workspace discovery, scanning, and canonical path sandbox |
| `tests/testdata` | Sanitized workspace fixtures only |
| `packaging` | Package-manager metadata |
| `scripts/assets` | Generated visual assets |
| `scripts/release` | Cross-platform build and checksum helpers |

## Runtime data boundary

The repository does not own the developer's `.boi` runtime directory. A
sanitized fixture lives at `tests/testdata/workspace/.boi`. Real provider keys,
memory entries, backup files, and machine-local configuration must remain
ignored.

## Migration rules

1. Preserve command names and defaults during Phase 1.
2. Move behavior behind `internal/app` before changing it.
3. Keep provider, tool, memory, and transport contracts independently testable.
4. Do not connect model output to process execution until the Phase 2 approval
   boundary exists.
5. Update this document whenever package ownership changes.

## Current TUI safety boundary

The TUI routes model requests through `internal/agent.Service`. Local
workspace inspection is currently explicit and read-only through
`/workspace`, `/ls`, and `/read`; model output cannot invoke these operations.
Write and process execution remain disabled until an approval state is visible
and testable in the TUI.

The workspace boundary validates lexical and canonical paths, including
symlink targets. It constrains filesystem paths only and must not be described
as process, container, or operating-system isolation.
