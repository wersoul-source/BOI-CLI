# Changelog

All notable changes to BOI CLI.

## v2.1.0 — Coding benchmark (unreleased)

- `process.run` reports a non-zero exit as a result (`exit status: N` with stdout and stderr) instead of an error. Before, a failing `go test` ended the task, because a failed command is never retried, and its stdout never reached the model
- `go run ./scripts/acceptance/codingbench -runs 3` runs four coding tasks (fix a bug, implement a function, edit a config file, rename across files) against the Provider in `PSC_1_*` under the `coding` preset. The host decides pass or fail by running the tests itself and checking that protected test files did not change; writes and commands are approved automatically, so the harness is never part of the `boi` binary

## v2.0.0 — Single SubAgent (unreleased)

Roadmap step 5 (ADR-003). Optional: no preset includes it.

- New pack `subagent` adds `agent.delegate(task, tools)`: the parent hands a self-contained sub-task to a child Agent that may use only the listed Tools
- The child never has more authority than the parent: only Tools the parent is offered, no further delegation, the parent's approval path (writes still ask the user; denied in `boi ask`), the parent's qualified Providers
- A delegation is classified from the Tools it grants: automatic when all are automatic reads, otherwise it needs approval
- Parent and child share one token and Tool-call budget; the engine checks it before every provider call
- The child sees only its task (no conversation, memory or Skills); its answer returns as an untrusted observation, and its changed paths go into the parent's manifest
- Requires a Provider qualified for Tool calling; `boi env` reports it like any Block
- Benchmark of delegated tasks (ADR-003 gate 6) is deferred

### Security and correctness (final audit)
- **MCP servers in a workspace `environment.json` no longer start until the user runs `boi env trust`.** Before, opening a cloned repository with `boi` ran any command its environment file named. Trust is stored per user, outside the repository, for the exact file content; any edit withdraws it
- The Agent never reads `.env` or `.env.*` files (templates such as `.env.example` excepted), and `workspace.search` skips them, so a prompt-injected task cannot pick up Provider keys
- MCP approval prompts show the exact arguments that will be sent
- Call IDs written by the model in text envelopes are replaced with host IDs; a copied example ID made approved commands replay an earlier result and could make `context.expand` return another call's output
- `boi` starts even when a required Block needs a qualified Provider that is not set up yet; the reason is shown instead
- `workspace.search` stops at its timeout and caps the entries it visits
- Granting `context.expand` to a SubAgent no longer fails; enum checks on array values no longer panic; temperature 0 is sent to OpenAI-compatible Providers; `{"path":"","paths":[...]}` reads every listed file; Provider error bodies are cut to 500 bytes
- MCP servers are told the real BOI version; `boi version` shows the real Go version

### Cleanup
- Removed about 1,200 lines of code nothing called (old compaction and context managers, Skill and Persona registries, streaming helper, unused TUI renderers and Thai string helpers)
- README, architecture docs and the winget manifest updated to v2.0; a stray worksheet file removed

```json
{"id": "subagent", "required": false}
```

## v1.11.0 — Presets

Roadmap step 4.

- `boi env presets` lists ready-made environments; `boi env use <preset>` writes one as `.boi/environment.json` (an existing file is kept unless `--replace`, which backs it up first)
- `minimal`: read-only, `workspace-read` (list, read, search); nothing is written or run
- `coding`: `workspace-tools` and `process-tools`, both required; writes and commands need approval
- New built-in pack `workspace-read`: the read-only part of `workspace-tools`
- No `research` preset yet: research needs web search, which BOI does not ship. Add an MCP search server to an environment instead
- `PRESET=<name> scripts/acceptance/benchmark.sh` runs the benchmark under a preset

## v1.10.0 — Composition report

Roadmap step 3.

- `boi env` (and `boi env --json`) composes the Agent exactly as `boi` and `boi ask` do and reports each Block from what the Runtime holds, not from the manifest:
  - `active`: at least one of its Tools is offered to the model
  - `loaded`: its Tools are registered, but none can be offered now (for example, no qualified Provider allows Tool calling)
  - `declared`: listed in `environment.json` but not loaded, with the reason (not installed, MCP server did not answer, ...)
- The report does not stop at the first problem: every Block is listed, and a missing required Block marks the Agent not ready (non-zero exit)
- Runtime Tools such as `context.expand` are listed separately
- Qualification now expires after 30 days; an expired Provider leaves the Router until `boi provider qualify` runs again. A change of model, Provider or endpoint already required a new qualification (it is part of the profile fingerprint)

## v1.9.0 — MCP Blocks

Roadmap step 2: capabilities can be added without building BOI.

- An `environment.json` entry with `mcp` starts a local MCP server at start-up, lists its tools and plugs them in as `mcp.<id>.<tool>`
- Every MCP Tool is external and needs the user's approval; its description and argument schema come from the server, and the Broker validates arguments against that schema
- `tools` limits which of the server's tools are plugged in
- A server that cannot start, does not answer within 15 s, or lacks an allowlisted tool is skipped when optional and stops start-up when required
- MCP servers start without `PSC_*` variables, so they never see Provider API keys
- MCP Tools join each task's working set while there is room under the 15-Tool limit

```json
{"schema_version": 1, "name": "with-github", "blocks": [
  {"id": "workspace-tools", "required": true},
  {"id": "github", "required": false, "mcp": {"command": "github-mcp-server", "args": ["stdio"], "tools": ["search_issues"]}}
]}
```

## v1.8.0 — Composable Blocks, step 1

- Block packs: a pack manifest declares what a pack provides, the runtime version and Provider capabilities it needs, and its highest allowed risk (ADR-002)
- `.boi/environment.json` chooses the packs of a workspace's Agent and marks each required or optional; without the file the Agent is composed exactly as in v1.7
- Composition is checked before any task: a missing or incompatible required pack stops start-up with the reason (`boi ask` exit 5, shown in the TUI); an optional one is skipped and reported
- Built-in packs: `workspace-tools` (list, read, search, write) and `process-tools` (`process.run`)

Example, an Agent that cannot run commands:

```json
{"schema_version": 1, "name": "no-shell", "blocks": [{"id": "workspace-tools", "required": true}]}
```

## v1.7.0 — Fewer turns, smaller context

Driven by the v1.6 benchmark on Cloudflare Workers AI: llama-3.3-70b never
issued parallel tool calls, and `deep_read` spent 21k-35k tokens per task.

Result (llama-3.3-70b, 6 tasks x 3 runs, same day): v1.6 17/18, 57 provider
calls, 191k tokens, 121 s; v1.7 18/18, 35 calls, 56k tokens, 50 s.
`deep_read` dropped from 28k-65k tokens to 2.5k-16k.

- New Tool `workspace.search(pattern, path)`: finds matching lines with file and line number in one file or a folder (skips `.git`, `.boi`, `agent-folder`, `node_modules`, binaries and symlinks; 50 matches max). In v1.6 the model paged through large files with `context.expand` instead of using `match`; a search Tool is the step models already know. Existing workspaces get it after adding it to `.boi/registry/tools.json` or re-creating the index
- `workspace.read` accepts `paths` (up to 5) to read several files in one call, so batching no longer depends on the model emitting parallel tool calls; a missing file is reported on its own instead of failing the batch
- Tried and reverted: shrinking the in-context preview of large outputs to 4 KB and `context.expand` to 8 KB. `deep_read` passed 1/3 (two runs hit the 64k token budget) against 2/3 for v1.6, because the model read more, smaller chunks

## v1.6.0 — Adaptive Execution

### Native tool calling
- Providers can carry native tool definitions (`llm.ToolCaller`); the OpenAI-compatible adapter sends `tools` and reads `tool_calls`
- Native mode is used only when every Router candidate supports it and the `native_tool_schema` probe proves the model calls tools correctly; otherwise the `<boi-action>` envelope remains, and an envelope is still accepted in native mode
- `port.ToolSpec` is the single tool contract; `ProviderSchema()` sends only a supported JSON Schema subset and `ValidateArguments` checks every call (native or envelope) before authorization. See `docs/architecture/ADR-001-native-tool-schema.md`
- Provider call IDs are replaced with host IDs, so Providers that reuse `call_0` cannot collide with idempotency keys

### Batch execution
- One turn may propose up to 5 Tool calls
- A batch made only of automatic reads runs concurrently; any change, process or external call keeps the batch sequential, and each call is still authorized, executed and verified on its own
- A denied or failed call stops the rest of the batch

### Context lifecycle (compute once, reference many)
- Every tool output is kept in a per-task context store with a checksum
- An output above 16 KB keeps its head and tail and names the reference and line range of the omitted middle; for `workspace.read` the file content is shortened inside the JSON, so line numbers refer to the file and fields such as `Truncated` stay intact
- When the 48 KB observation budget is spent, older observations become stubs with their reference instead of disappearing
- New Runtime Tool `context.expand(ref, lines)` reads any stored output back exactly, after an integrity check; it is added to every non-empty active set that has room
- No lossy compression: nothing is summarized or abbreviated, so an answer built from expanded context sees the original text

### Planning
- The static Task Plan is no longer sent to the model every turn; it stays in the task manifest

### Qualification
- Probes judge behavior instead of exact text: prose around a correct answer passes, a forbidden outcome fails, an empty reply is `inconclusive` and enables nothing
- `native_tool_schema` is now measured: `unsupported` (adapter or endpoint), `failed` (model ignores tools) or `passed`
- Probe suite is `boi-provider-v2`: **run `boi provider qualify <name>` again after upgrading**

### Measurement
- `scripts/acceptance/benchmark.sh` runs a fixed read-only task set against a real Provider and can compare two versions (`BASE_REF=...`)

## v1.5.0 — BOI Agent Suite v1.5

### Architecture
- Fixed spine (`core`, `runtime`) and pluggable Blocks (`equipment`, `service`, `agentfolder`, `subagent`)
- `block/port.Tool` contract; the Broker dispatches through a registered tool table
- Built-in workspace, process and MCP Tools moved to Equipment and verify their own effects
- `app.BuildAgent` is the single composition point for TUI and CLI
- Dependency rules enforced by `internal/architecture/deps_test.go`

### Fixes
- Provider adapters: request timeout, 8 MB response cap; Google API key moved from URL to header
- Memory store rejects path-traversal IDs; persona names match case-insensitively
- Upgrade caps extracted archive entries at 512 MB
- Command deny-list hardened and documented as best-effort, not isolation
- `doctor` reports Config and Agent identity as separate checks

### Build
- Linux is the release gate: race tests, staticcheck, coverage floor, built-binary folder simulation
- Other platforms are compile checks; `make smoke` runs the Linux acceptance on a VM

## v0.3.0 (2026-08-03) — Cross-platform Release + Full Audit

### Cross-platform Support
- Build for **Windows** (amd64, arm64), **Linux** (amd64, arm64), **macOS** (amd64, arm64)
- Refactored `internal/term/` with Go build tags — no more Windows-only imports
- `SetUTF8Console()` now a no-op on Unix (UTF-8 is default)
- `EnsureThaiFont()` now a no-op on Unix (system fonts available)

### Bug Fixes
- **Wizard keyboard**: Fixed — textInput.Update() was never called in stepAPIKey
- **Doctor false-positive**: Fixed — "All checks passed!" printed even on fail
- **Doctor count mismatch**: Fixed — counted 4 providers, factory supports 20
- **Font error handling**: Fixed — AddFontResourceW return now checked
- **Version override**: Fixed — changed `Version` from `const` to `var` so ldflags works

### Infrastructure
- Release v0.3.0 includes binary archives for all 6 platforms + checksums
- `install.sh` (Linux/macOS) auto-fetches latest release from GitHub
- `install.ps1` (Windows) auto-fetches latest release (no more hardcoded version)
- Full audit of 70+ source files — 4 bugs found and fixed

## v0.2.0 (2026-08-01) — Thai Terminal Fix + PSC Rotation
- Thai terminal rendering fix (codepage 874 → UTF-8, ThaiStringWidth, font install)
- PSC (Provider Supply Chain) auto-rotation with usage bar
- TUI improvements: responsive input, Thai-safe chat bubbles

## v0.1.2 (2026-07-31) — Provider Test + Setup Wizard
- Provider test after setup — validates API keys immediately
- Complete onboarding wizard

## v0.1.1 (2026-07-31)
- Bug fixes and improvements

## v0.1.0 (2026-07-31) — Initial Release
- Core CLI with TUI, 6 personas, Phantom DB, PSC rotation
