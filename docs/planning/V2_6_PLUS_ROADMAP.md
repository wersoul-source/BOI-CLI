# V2.6+ Roadmap — capability gaps found at v2.6.2

Owner's eight asks, each mapped to what exists, what is missing, and the
smallest safe first step. Everything below respects the runtime contract
(`docs/architecture/AGENT_RUNTIME_CONTRACT.md`): model output is untrusted
intent; the host authorizes, acts and verifies.

## Tier 1 — small, fits the current architecture

### 1. File delete (`workspace.delete`)

- Exists: `workspace.write`, `workspace.edit`, `workspace.list/read/search`.
- Missing: no delete Tool at all.
- Step: new Tool in `internal/equipment/tools/filesystem` (`Risk: change`,
  `Approval: confirm`, resolve through `workspace.Sandbox.ResolveExisting`,
  refuse if the resolved path is the sandbox root or a `.boi` control path;
  report `ChangedPaths` so the effect ledger counts it). Register it in the
  same pack as the other filesystem Tools and add broker/engine tests.

### 2. Skill authoring by the Agent

- Exists: Skills are plain `.skill.md` files; Skill packs plug into
  `environment.json`; the Agent already has `workspace.write`.
- Missing: no flow that turns research or a correction into a saved Skill.
- Step: an active Skill "skill-author" (text instructions only, no authority)
  that tells the model the file format and where drafts belong
  (`agent-folder/bin`), plus a `boi skill promote <draft>` command that
  validates and moves the file into `.boi/skills/`. The Agent never edits
  registry files directly; the user promotes.

## Tier 2 — design work, medium

### 3. Tools of every kind

- Exists: Tools are Go implementations behind `port.Tool`, composed via
  block packs; MCP is the sanctioned way to add capabilities without Go.
- Missing: runtime-created Tools (a model cannot compile Go, and should not).
- Step: do not build dynamic Tool compilation. Instead: (a) document MCP as
  the extension path, (b) add an MCP "attach on command" flow (see 4), and
  (c) let the Agent draft an MCP server project as files for the user to run.

### 4. MCP connect on command

- Exists: MCP blocks in `environment.json` (after `boi env trust`), client
  in `internal/equipment/tools/mcp`.
- Missing: adding an MCP server mid-task (requires recomposing the
  environment, which is a start-of-task operation today).
- Step: `boi env add-mcp <name> <command>` CLI that edits
  `environment.json`, validates with a probe, and asks for trust; then the
  Agent can tell the user to run it, or request it via a host-approval
  panel. Mid-task hot-attach to a running Engine is a later, riskier change.

### 5. Plugin compatibility layer

- Exists: the Block/Environment pack system is the plugin system.
- Missing: nothing cross-language except MCP.
- Decision needed from the owner: whether a general plugin loader is wanted
  at all, or MCP + packs are enough. Recommend: enough.

## Tier 3 — larger projects

### 6. Session history and work splitting

- Exists: Agent Folder task records and manifests persist per task; the
  SubAgent Block exists (`internal/subagent`, ADR-003) with a shared budget
  and shared effect ledger.
- Missing: resumable conversation sessions; a way to split work into several
  named sub-sessions from the TUI.
- Step: persist the in-memory `conversation` slice as an append-only
  transcript under `agent-folder/`, add `boi session list/resume <id>` and a
  TUI command. Sub-session splitting can build on the existing SubAgent
  delegation rather than new plumbing.

### 7. Web UI on the TUI

- Exists: proposal only (`docs/planning/BOI_WEB_UI_PLAN.md`).
- Step: read the plan, decide between (a) embedded webview overlay or
  (b) local web server + browser, then write an ADR before any code.

### 8. Running `boi` commands from a session

- Exists: the `process` Tool runs shell commands, so `boi ask ...` from
  inside the TUI works once the process pack is active and approved.
- Missing: nothing structural; document it, and consider a `boi.run` Tool
  that invokes the CLI in-process (no shell) with a fixed argument schema.

## Order of attack

1. `workspace.delete` (Tier 1.1) — one Tool, closes the owner's ask 1.
2. Skill authoring flow (Tier 1.2) — closes ask 2.
3. `boi env add-mcp` (Tier 2.4) — closes asks 3–5 pragmatically.
4. Session persistence (Tier 3.6) — closes ask 6.
5. Web UI ADR (Tier 3.7) — starts ask 7.

Each step ships as its own release with tests in
`internal/runtime/agent` (engine/broker) and the relevant equipment package.
