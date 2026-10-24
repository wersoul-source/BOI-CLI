# Changelog

All notable changes to BOI CLI.

## v2.6.1 — Add Maxplus.ai (unreleased)

- **Maxplus AI** (`maxplus`, `https://api.maxplus-ai.cc`) is in the provider list. Its models come from the live list, supporting both OpenAI-compatible and Anthropic-compatible endpoints

## v2.6.0 — Providers that stay current (unreleased)

- **The wizard asks the provider for its models.** After the key is entered, the wizard reads `GET <address>/models` and offers that list, so it is never older than the provider; the built-in list is the fallback when the server cannot be asked (Anthropic and Google use other protocols and keep the built-in list). Custom endpoints get the list too, with "Type custom model" still available
- **OpenCode Zen** (`opencode`, `https://opencode.ai/zen/v1`) is in the provider list. Its models come from the live list. Models Zen serves through other routes than chat completions (some Qwen and Gemini models) are not covered
- A test requires every provider in the catalog to be buildable by the factory

## v2.5.2 — Wizard: typed model names, working TUI commands (unreleased)

The second real hosted run saved the model `custom` and an address cut short, and every check failed with 404 and then 400.

- **A model name typed in the wizard never arrived.** The model step threw away every key press, and an empty name was quietly replaced by the word `custom`, so a provider with a typed model (every custom model, and the "Custom model" item of every provider) was saved with the model `custom`. Typed keys now reach the text box, and an empty model name is refused with a message instead of replaced. This bug is older than v2.5; the v2.5.1 tests set the text box directly and missed it, so the wizard tests now type key by key and a typed and a pasted 84-character address are checked to arrive whole
- `boi quickstart` shows the address it saved next to the model (`Address: https://…/ai/v1`, without any user name or query), so a missing character can be seen before the check
- `boi quickstart --change` chooses again: a saved model was kept and re-checked every time, so a wrong address or model could not be corrected without editing `.env`. The saved values are forgotten first, because a value already read from `.env` is never replaced by a later read
- **TUI commands**: `/model` and `/config` were offered in the palette but answered "Command not recognized", and `/model NAME` said "Model set to" without changing anything. `/model` now shows the model, provider and address in use, `/config` shows the folder and model, `/model NAME` says honestly that the model is chosen with `quickstart --change`; a test requires every command in the palette to be recognised
- **TUI without a model**: the message tells a missing model apart from a configured one that has not been checked in this folder, names the folder, and uses the command you actually ran (`boi-agent`, not `boi`). `/provider` and `/config` give the same guidance
- When every check gets 404 the check prints the address and the model it used and what to look for; when every check gets 401 it still names the server

## v2.5.1 — Setup wizard fixes (unreleased)

Found when the first real hosted run failed every check with HTTP 401.

- **The wizard saved the model name as the API key** whenever a model was typed instead of picked from the list. The text box was shared between the key and the model name. The key is now kept apart; going back to the key step shows the key again
- **"Other (custom)" never worked.** It saved a provider named `other`, which the factory does not know, so it was silently dropped. It now asks for an OpenAI-compatible address (Cloudflare Workers AI, a server on your network, a gateway), then the key and the model, and saves an `openai` provider with that address. A key was always sent to the address in the catalog (OpenAI's), so a Cloudflare key failed with 401
- `boi quickstart` uses a single-provider wizard: the "how many providers" question is gone from that path
- The check prints the server it talked to (`Endpoint: api.openai.com`, host and port only) and, when every probe is rejected with 401, says that the server did not accept the key and what to check
- The wizard tests drive the real key presses: custom endpoint, address validation, Esc at each step, a typed model name, a local endpoint without a key
- `GET_STARTED.md` explains that a key only works on its own provider's server and how to give Cloudflare's address

## v2.5.0 — Easier first run (unreleased)

Using BOI took a Go toolchain, six commands in the right order, and knowing the environment variables. v2.5 gives it one guided step and a real download.

- New `boi quickstart`: finds a model server on this computer (Ollama, LM Studio, llama.cpp on their usual ports) and lists its chat models to pick with Enter; or offers a hosted provider (the existing wizard) or a server at another address; saves the choice in `.env` (kept out of Git); checks the model; and says in plain words what the Agent can now do. Run again to change the model; a model already checked is not checked again
- The first launch of `boi` in a project does the same instead of the arrow-key wizard, and asks to check the model with the right timeout (3 minutes per step for a local model, which used to get 30 seconds on this path) and without calling a free local model "paid"
- README starts with a three-step Quick start (install, `boi quickstart`, `boi`) for Linux and Windows PowerShell; `GET_STARTED.md` is trimmed to match
- Install scripts: the PowerShell script pointed at a branch that does not exist (`main`; the branch is `master`) and the shell script at a domain that is not ours; both are fixed. The PowerShell script picks the ARM64 build on ARM64 Windows and no longer runs `boi init` in whatever folder it was started from. Both end with the two commands to run next
- Release builds are Linux and Windows (amd64, arm64); macOS is no longer built. Pushing a `v*` tag publishes them (`.github/workflows/release.yml`)

## v2.4.0 — Skill packs (unreleased)

ADR-004. Skills and Tools are independent Blocks; until now only Tools could be chosen in `environment.json`.

- An `environment.json` entry with `skills` plugs in a folder of `*.skill.md` files (relative to `.boi`): `{"id": "team-skills", "required": false, "skills": {"dir": "skills/team"}}`
- The folder must stay inside `.boi` after symlinks are resolved; entries must be plain files of at most 64 KB, and at most 15 Skills. A missing or empty folder, a file that does not parse, or a Skill name already provided by another pack refuses the pack (required stops start-up, optional is skipped and reported)
- Pack Skills join the registry's Skills when a task selects its working set. A Skill named in `.boi/registry/skills.json` wins over a pack Skill of the same name. A Skill is offered with or without the Tools it names; the model decides whether to load it
- A Skill is untrusted text and adds no Tool or authority, so a Skill pack needs no `boi env trust`
- `boi env` lists a skill pack's Skills: `active` when a qualified Provider allows Skills, `loaded` when it does not, `declared` when refused
- No built-in skill pack and no preset Skill yet: one ships when a benchmark shows it helps

Reliability and security (same release, ADR-005):
- **Skill files can no longer be read from outside `.boi/skills`.** A registry entry (for example in a cloned workspace) could name a symlink to any file the user can read and the file's text reached the model. Skills now load through `skill.LoadConfined`: after symlinks are resolved the file must be a regular file inside its folder and at most 64 KB
- The host records what a task really did: `Outcome` (`answered`, `no_effect`, `executed`, `changed`) and `ChangedPaths`, in the result and in `boi ask --json` (`outcome`, `changed_paths`). An effect is a verified Tool call that is not a read and not a delegation; SubAgent children count through the shared task ledger, so a delegation alone proves nothing
- The v2.3 challenge now rests on that evidence instead of the Tool-call count: an answer about a workspace file with no effect is sent back once when the model made no Tool call, only delegated, had every change fail, or offered code in a fenced block as text. Reading and then answering in prose is left alone. The notice reads "Nothing was changed or run in this task."
- `codingbench` prints `stop/outcome` per run and counts false completions

Local models (same release):
- A Provider on `localhost`, `127.0.0.1` or `[::1]` needs no API key, in the environment, in `.env` and in `boi setup` (an empty key used to skip the Provider). `PSC_1_NAME=ollama` without a base URL now means `http://localhost:11434/v1` instead of OpenAI's address, and no `Authorization` header is sent without a key
- Task limits can be set with `BOI_TASK_TIMEOUT`, `BOI_MAX_TOKENS`, `BOI_MAX_TOOL_CALLS` and `BOI_MAX_STEPS`, and the per-request timeout with `BOI_REQUEST_TIMEOUT`. The defaults (2 minutes, 64,000 tokens) stay. An invalid task limit stops start-up and names the variable; an invalid request timeout is ignored
- A local server gets 3 minutes per qualification probe instead of 30 seconds, because its first answer loads the model
- `GET_STARTED.md` rewritten for v2.4: source install, Provider setup for hosted and local models in PowerShell, qualification, environments and troubleshooting
- `codingbench.sh` runs against a local server too; its per-task limit follows `BOI_TASK_TIMEOUT`
 (ended `completed`, host check failed)

## v2.3.0 — No silent "done" (unreleased)

The coding benchmark showed llama-4-scout answering "fixed" in 10 of 12 runs without calling a single Tool; the run ended as `completed` and a user would have believed it.

- In an interactive run where the model can change files or run commands, an answer that names an existing workspace file (in the task or in the answer, any language) but follows no Tool call is sent back once: "you answered without using any Tool, so nothing was changed or run..." The model then acts or answers again
- If it answers again without a Tool, the run still completes, with `Notice`: "No Tool ran in this task: no file was changed and no command was run.", shown under the answer in the TUI
- Plain conversation (no workspace file named), non-interactive runs (`boi ask`, `--json`) and read-only environments are never challenged, so they cost no extra provider call
- `codingbench` marks a challenged answer as `answer?` in the Tool trace

## v2.2.0 — Edit in place, Skills always offered (unreleased)

Driven by the first coding benchmark on Cloudflare Workers AI (llama-3.3-70b, 5/12): every failed `implement` and `edit_config` run rewrote a whole file to change one line and lost or invented the lines around it (a changed `package` name, config keys dropped or made up).

- New Tool `workspace.edit(path, old, new)` replaces one exact piece of text in an existing file; `old` must appear exactly once, so every other byte stays as it was. It keeps the file mode, matches Windows line endings, refuses credential files and needs approval like `workspace.write`. `workspace.write` now says it is for new files or whole-file replacement. Existing workspaces get the Tool without editing `.boi/registry/tools.json`
- A Tool call that failed before changing anything (edit text not found or not unique, command blocked by the sandbox) goes back to the model to correct instead of ending the task. Before, the first `workspace.edit` whose `old` text did not match stopped `edit_config` after one call in all three Cloudflare runs. Tools mark such errors with `port.NoEffect`; a failure after a change has started still stops the task
- `codingbench` prints the order of Tool calls for each task (`read>edit!>edit>process.run`, `!` = failed)
- `bash scripts/acceptance/codingbench.sh [MODEL...]` runs the coding benchmark on one or more Cloudflare models and asks once for the account ID and token (`CF_ACCOUNT_ID` and `PSC_1_*` skip the questions); it ends with one result line per model
- Skills and Tools are independent: a Skill is offered whether or not the Tools it names are active (its description then says which are missing), and keyword overlap with the task only orders Skills instead of hiding them, so a Thai task no longer loses every English-tagged Skill. The model decides whether a Skill applies; a task needs neither a Skill nor a Tool to run

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
