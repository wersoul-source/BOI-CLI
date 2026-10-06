# BOI Block Architecture

This is the source of truth for package ownership. It replaces the Phase 1
layout in `STRUCTURE.md`. A dependency test (`internal/architecture`) enforces
the rules below. A change that breaks them fails `go test ./...`.

## Principle

BOI has a **fixed spine** and **pluggable Blocks**.

- The spine (Core + Runtime) defines identity, policy, lifecycle and the
  contracts (ports) every plug-in must satisfy. It changes rarely and never
  imports a plug-in.
- Blocks plug into the spine by implementing its ports. A Block can be
  replaced or removed without editing the spine.
- `internal/app` is the only place that knows which Blocks exist and wires
  them into the spine.

Go cannot load compiled plug-ins at runtime on every target, so BOI has
exactly two plug paths:

1. **Compile time**: a package implements a spine port and `internal/app`
   registers it.
2. **Run time**: an external process speaks MCP and is adapted into a Tool by
   `equipment/tools/mcp`. Nothing external is linked into the binary.

## Layout

```text
cmd/boi                      executable entry point
internal/
  block/                     shared vocabulary: Block IDs and Manifests (leaf)
  block/port/                plug-in contracts: Tool, ToolSpec, ToolResult (leaf)

  core/                      Block 2 Core        FIXED  identity, qualification, environment policy
  core/persona/                                  FIXED  embedded boi Persona
  runtime/                   Block 4 Runtime     FIXED  engine, broker, approval, service, lifecycle
  runtime/llm/                                   FIXED  Provider port, Router, error classes
  runtime/workspace/                             FIXED  workspace path boundary

  equipment/                 Block 3 Equipment   PLUG   manifest
  equipment/tools/workspace/                     PLUG   workspace.list/read/write
  equipment/tools/process/                       PLUG   process.run
  equipment/tools/mcp/                           PLUG   MCP client + Tool adapter
  equipment/registry/                            PLUG   15/15 bounded capability index
  equipment/skill/                               PLUG   Skill files
  equipment/memory/                              PLUG   memory store and weights
  service/                   Block 1 Service     PLUG   manifest
  service/provider/                              PLUG   adapters, catalog, factory
  service/config/                                PLUG   config and .env loading
  agentfolder/               Block 5 AgentFolder PLUG   bin/output tray, manifests
  subagent/                  Block 6 SubAgent    GATED  disabled until evaluation gate

  app/                       composition root (the only wiring point)
  transport/cli, tui         input and rendering
  platform/                  OS glue: terminal, logging, update
  acceptance/                built-binary acceptance tests
```

## Dependency rules (enforced)

| From | May import | Must not import |
|---|---|---|
| `block`, `block/port` | standard library | any BOI package |
| spine (`core`, `runtime`) | spine, `block`, `block/port`, `platform` | any plug Block, `app`, `transport` |
| a plug Block | spine, `block`, `block/port`, `platform`, its own subpackages | another plug Block, `app`, `transport` |
| `app` | everything except `transport` | `transport` |
| `transport` | everything | (target: only `app` and spine types; tracked as a shrinking allowlist) |

## Ports

| Port | Owner | Implemented by |
|---|---|---|
| `port.Tool` | `block/port` | `equipment/tools/*` |
| `llm.Provider` | `runtime/llm` | `service/provider/adapters` |
| `runtime.TaskRecorder` | `runtime` | `agentfolder` |
| `runtime.MemoryHook` | `runtime` | `equipment/memory` |
| `runtime.SkillDoc` | `runtime` | built by `app` from `equipment/skill` |

### Adding a Tool

1. Create a package under `equipment/tools/` that implements `port.Tool`.
2. Register it in `app.BuiltinTools` (or adapt it through MCP).
3. Add an index entry in `app.DefaultCapabilityIndexes`.

The Broker, Engine and Service are not edited.

## Delivery order

| Step | Change | Done when |
|---|---|---|
| A1 | Move packages into the layout above | build and all tests green, no behavior change |
| A2 | Introduce `port.Tool`; Broker dispatches through a tool table | Broker has no tool-specific `switch` |
| A3 | Move built-in tools and MCP adapter to Equipment | Runtime imports no tool implementation |
| A4 | Runtime uses `MemoryHook` and `SkillDoc` ports | Runtime imports no plug Block |
| A5 | One composition function in `app` used by TUI and CLI | Service is built in one place |
| A6 | Dependency rule test | rules above fail the build when broken |
| A7 | Linux first: CI gate on Linux, smoke script for a Linux VM | `scripts/acceptance/linux_smoke.sh` passes on the VM |

Windows, macOS and Android remain compile checks only until Linux passes on
a real machine.
