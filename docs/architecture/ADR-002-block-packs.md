# ADR-002: Block packs and environment manifests

Status: accepted for v1.8 (roadmap step 1). Scope: `block`, `block/port`,
`app`. Core and Runtime unchanged.

## Context

v1.7 composed every Agent from one hard-coded Tool list (`app.BuiltinTools`).
Tools could be switched off per workspace through the capability index, but
there was no unit larger than one Tool, no way to say a capability is
required, and nothing checked a pack's needs before a task started.

## Decision

1. **Pack manifest** (`block.PackManifest`, data only): id, kind, version,
   `provides`, `requires` (minimum runtime, qualified Provider capabilities),
   `max_risk`. Only `tool-pack` exists; other kinds are added when a real
   Block of that kind ships (MCP is next).
2. **Environment manifest** (`.boi/environment.json`): names the packs and
   marks each required or optional. No file means the default environment,
   which composes exactly the v1.7 Tool set.
3. **Resolver** (`app.Compose`) runs before any task:
   - pack not installed, runtime too old, Provider not qualified for a
     required capability, a Tool above `max_risk`, `provides` not matching
     what the pack builds, or a Tool name owned by another pack → the pack
     is refused;
   - refused and required → start-up stops with the reason (`boi ask` exit
     5 `unavailable`; the TUI shows it);
   - refused and optional → skipped and listed in `Agent.Composition`.
4. **Authority only narrows.** `max_risk` can refuse a Tool; nothing in a
   manifest can raise a Tool's risk or relax its approval. `port.ToolSpec`
   stays the source of truth (ADR-001).
5. **Built-in packs:** `workspace-tools` (list, read, search, write; required
   by default) and `process-tools` (`process.run`; optional).

## Consequences

- Leaving `process-tools` out of the environment gives a read-and-write Agent
  that cannot run commands, without code changes.
- An invalid `environment.json` is an error, never a silent fallback to the
  default.
- Composition happens at start only. Changing the file affects the next run.
- Packs are compiled in. Loading packs from outside the binary arrives with
  the MCP Block (step 2), which needs no Go code per server.

## Rollback

Revert the PR. Workspaces without `environment.json` behave exactly as v1.7;
a workspace that added the file must delete it.
