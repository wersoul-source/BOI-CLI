# Platform principles BOI should prepare for (2026-11-03)

Status: **research note**. It is not a decision. Inputs for Architecture IV
([ADR-006](../architecture/ADR-006-calibration.md),
[ARCHITECTURE_IV.md](../architecture/ARCHITECTURE_IV.md),
[renovation plan](../planning/ARCHITECTURE_IV_RENOVATION.md)).

Purpose: this note does not rank the platforms. It lists concepts other agent
platforms have that BOI lacks. For each concept it decides whether BOI must fix
a contract **now**, so that adding the concept later does not rewrite Core.
BOI's identity stays as it is: six Blocks, a small fixed Core, a detachable
periphery, calibration with the four statuses `NATIVE` / `ASSISTED` /
`UNVERIFIED` / `BLOCKED`, and measurement before trust.

Legend for evidence: **[V]** read on an official page in this session ·
**[S]** seen only in a search snippet or a secondary source, so it is
*unverified* · **[U]** inferred, not confirmed.

---

## 1. Scope and sources

The research was shallow: principles only, with no implementations and no UI.

| Source | Reached? | Used for |
|---|---|---|
| `code.claude.com/docs/en/hooks`, `/checkpointing`, `/skills`, `/sub-agents`, `/memory` | **yes** (fetched) | Claude Code [V] |
| `code.claude.com/docs/*/glossary` | search snippet only | permission modes [S] |
| `github.com/sst/opencode` README; `packages/web/src/content/docs/{permissions,server,lsp}.mdx` on the `dev` branch | **yes** (fetched via github.com) | OpenCode [V] |
| `opencode.ai` | **no** (DNS blocked in sandbox) | — |
| `github.com/openai/codex` README, `docs/sandbox.md` | yes, but both only point to external docs | — |
| `developers.openai.com/codex` | **no** (DNS blocked) | Codex claims are all [S] |
| Secondary Codex guides (devhints, blakecrosley.com, codex.danielvaughan.com) | search snippets | Codex [S] |
| `cursor.com/docs` | **no** (DNS blocked); search snippets of its pages only | Cursor [S] |
| `github.com/cline/cline` README | **yes** (fetched) | Cline [V] |
| `docs.cline.bot` | **no** (DNS blocked); search snippets only | Cline hooks/checkpoints [S] |
| GitHub API (`gh api`) for these repositories | **no** (403, repo not attached) | — |

---

## 2. Principles per platform

### Claude Code
1. **Deterministic lifecycle hooks, outside the model.** Events cover the session, the turn, the Tool call, permission, subagents, compaction and model switches. A hook can allow, deny or ask, rewrite input, or add context. Its input is JSON that carries `session_id`, `permission_mode` and `agent_id`. [V] [hooks](https://code.claude.com/docs/en/hooks)
2. **A checkpoint at every user turn.** The code, the conversation, or both can be restored separately. Checkpoints persist with the session, so they survive a resume, and they have a retention period. The docs state the limits plainly: shell-only edits, subagent edits and external changes are not tracked. [V] [checkpointing](https://code.claude.com/docs/en/checkpointing)
3. **Progressive disclosure for skills.** Only the description stays in context; the body loads when the skill is invoked. Skills are scoped (enterprise > personal > project, plugins namespaced) and can run in a forked subagent. [V] [skills](https://code.claude.com/docs/en/skills)
4. **A subagent has its own context, its own Tool allowlist and an optional model override.** The parent's permission mode caps the child. A background child raises its approvals in the main session, labelled with its name. Children can be addressed by id and resumed. [V] [sub-agents](https://code.claude.com/docs/en/sub-agents)
5. **Memory is context, not enforcement.** Instructions written by people are kept apart from memory the agent writes. Memory is scoped (org/user/project), loading is size-capped, and anything that must be enforced goes in a hook. [V] [memory](https://code.claude.com/docs/en/memory)
6. **Permission modes are named and apply to the whole session** (default, acceptEdits, plan, auto, dontAsk, bypassPermissions). [S] glossary snippet; the mode names are corroborated by [sub-agents](https://code.claude.com/docs/en/sub-agents) [V].

### OpenCode
1. **Client/server.** One headless server owns state, model calls and Tool execution. The TUI, IDE plugins and the SDK are clients. The server publishes an OpenAPI 3.1 spec and SSE event streams. [V] [server.mdx](https://github.com/sst/opencode/blob/dev/packages/web/src/content/docs/server.mdx)
2. **The session is an addressable resource.** It can be aborted, reverted to a message (and un-reverted), forked at a message, and asked for its child sessions. [V] same page
3. **Permissions are `allow`/`ask`/`deny`** per Tool, with pattern rules and per-agent overrides. A *doom loop* (the same call three times) → `ask`. A path outside the project → `ask`. [V] [permissions.mdx](https://github.com/sst/opencode/blob/dev/packages/web/src/content/docs/permissions.mdx)
4. **Primary agents (build, plan) and subagents.** `plan` is read-only by default. [V] [README](https://github.com/sst/opencode)
5. **LSP diagnostics as feedback for the agent.** Opt-in. [V] [lsp.mdx](https://github.com/sst/opencode/blob/dev/packages/web/src/content/docs/lsp.mdx)
6. **Provider facts come from an external catalogue** (models.dev). Snapshots live in a separate internal git store, never the project's history. Compaction is automatic at a token threshold. [S] secondary sources only.

### OpenAI Codex (every item unverified: the official docs were unreachable)
1. **Two independent controls:** the sandbox mode (read-only / workspace-write / full access) and the approval policy (untrusted / on-request / never). [S] [devhints](https://devhints.io/codex), [guide](https://blakecrosley.com/en/guides/codex)
2. **The cloud runs in isolated containers with network off during the agent phase.** Locally, an OS-level sandbox. [S] same
3. **The session is an append-only JSONL log of events.** Its event types match the `exec --json` stream. Resume and fork rebuild state from the log, and an ephemeral mode opts out. [S] [codex.danielvaughan.com](https://codex.danielvaughan.com/2026/04/29/codex-cli-rollout-files-session-recording-replay-audit-trails/)
4. **Layered instruction files** (`AGENTS.md`: global → repo → current directory) are merged. [S]
5. **A non-interactive mode with a machine-readable event stream** (`codex exec --json`). [S]

### Cursor (every item unverified: only search snippets of cursor.com pages)
1. **Checkpoints are taken automatically before significant changes.** They are stored locally, apart from Git, and can be previewed, then restored. [S] [history](https://cursor.com/docs/agent/chat/history)
2. **Summarization when context nears its limit.** A page exists; its content was not read. [U] [summarization](https://cursor.com/docs/agent/chat/summarization)
3. **Cloud agents run in isolated VMs from a declared environment** (repositories, dependencies, secrets, startup commands, network). [S]
4. **Hooks at project, user and enterprise levels.** Some hooks are unsupported in the cloud, so behaviour differs by environment. [S] [hooks](https://cursor.com/docs/hooks)
5. **Rules are retrieved by the agent by type and description**, not always injected. [S]

### Cline
1. **Plan mode versus Act mode.** In Act, every edit and command needs approval unless auto-approve is on. [V] [README](https://github.com/cline/cline)
2. **Every change is checkpointed**, which is the stated safety net that makes auto-approve acceptable. [V] README; [S] [checkpoints](https://docs.cline.bot/features/checkpoints)
3. **The agent watches linter and compiler errors** and fixes what they report. [V] README
4. **Hooks** (TaskStart, TaskResume, PreToolUse, …) read JSON on stdin and write JSON to stdout. Global hooks run before project hooks, and either can cancel. "PreToolUse validates; it does not decide." [S] [hook reference](https://docs.cline.bot/features/hooks/hook-reference)
5. **"Memory Bank" is a documented convention of files, not a runtime feature.** [S] [memory bank](https://docs.cline.bot/prompting/cline-memory-bank)
6. **The headless CLI defaults to Act with auto-approve.** BOI's D2 deliberately does the opposite. [S]

---

## 3. Gap table

Tiers: **P0**: lock the contract now (data shape, event, identity, security
boundary, versioning). **P1**: define the interface only, with no
implementation. **P2**: later, and it must not touch Core.

| # | Area | Principle seen elsewhere | BOI today | Tier | Owner Block | Phase |
|---|---|---|---|---|---|---|
| 1 | Agent identity, lifecycle, multi-agent | Addressable sessions and children, parent caps child (CC, OC) | `AgentPhase` FSM `runtime/agent/lifecycle.go`; `PlannedStep.DependsOn`; `Service.RunChild` + `sharedBudget` (`delegation.go`); **no agent/parent id in events, no messaging** | **P0** ids (`task_id`, `agent_id`, `parent_id`) · P2 messaging/teams | Runtime, SubAgent | 0 (Event v0), 3 |
| 2 | Background / scheduled work, cancellation | Background subagents, abort endpoint (CC, OC) | `StopCancelled`, `context` cancellation, TUI ctrl+c; no detached task | P1 detach/cancel interface · P2 scheduler | Runtime, Transport | 3 |
| 3 | Human in the loop (pause/resume/reject/timeout) | Approvals surfaced across agents; hooks may `ask`/`deny` (CC, Cline) | `ApprovalRequest` (states, `ExpiresAt`, call fingerprint) in `contracts.go`; `StopNeedsApproval`; D1/D2. **A pending approval does not survive a process exit** | **P0** approval as a persisted event bound to fingerprint + Recipe version | Runtime (Broker), Agent Folder | 2–3 |
| 4 | Session persistence, schema migration, event replay | Append-only event log, resume/fork from it (Codex [S], OC) | `agentfolder/store.go` checkpoint + manifest, `schema_version: 1`; `TaskRecorder` **must not persist prompts or model output**; `EngineEvent` has **no schema version or sequence** | **P0** event envelope + migration rule + transcript policy (owner decision) | Agent Folder, Runtime | 0, 3 |
| 5 | Long-term memory scope and retention | Scoped memory, human versus agent-written, size cap, "context not enforcement" (CC) | `equipment/memory` Store with `TTL`, `SessionID: "auto"`; no scope or provenance | P1 add `scope` + `source` fields | Equipment | 4 |
| 6 | Context engineering | Compaction/summarize with markers; retrieval on demand (CC, Cursor [U], OC [S]) | `context_store.go` (refs + `context.expand`); Recipe `context_policy` planned; no compaction | P1 Context Manager interface · its event kind is P0 (in #4) | Equipment, Core (Recipe) | 3 |
| 7 | Multimodal content | — (all accept images [U]) | `llm.Message{Role, Content string}` | **P0** typed content parts (text only implemented) | Service (Connector), Core (capability) | 1 |
| 8 | IDE integration / diagnostics | Server + clients over a published API; LSP diagnostics fed back (OC, Cline) | Transport in-process; no diagnostics | P1 a Transport-neutral command and event API · P2 LSP as Equipment | Transport, Equipment | 4–6 |
| 9 | Distributed task identity, lease, heartbeat | Cloud tasks (Codex [S], Cursor [S]) | Task id in `agentfolder`; `ToolCall.IdempotencyKey`, `IdempotencyKeyHash`; no lease | **P0** globally unique task id + reserved `owner`/`lease` fields · P2 lease logic | Runtime, Agent Folder | 3 |
| 10 | Sandbox backend | Sandbox mode independent of approval policy; OS isolation (Codex [S]) | Concrete `workspace.Sandbox` (paths) and `process.Sandbox` (deny-list); D7 | P1 `SandboxBackend` interface that reports its guarantees · impl Phase 4 | Runtime (Trusted Foundation) | 2–4 |
| 11 | Extension versioning and compatibility | Namespaced plugins, scoped precedence (CC) | `block.PackManifest` (`schema_version`, `version`, `requires.runtime/capabilities`, `max_risk`) | **P0** declare target contract versions, namespace, rights | Equipment (Extensions) | 0 → 4 |
| 12 | Observability and eval | Machine-readable event stream (Codex [S], OC SSE) | `AgentStep` trace, manifest usage/evidence; codingbench planned | **P0** correlation ids in every event · P1 metrics sink · P2 exporters | Runtime, Agent Folder | 0, 5 |
| 13 | Provider failover, affinity, capability negotiation | Model switch hooks (CC); per-agent model (CC, OC) | `llm/router.go` failover by error class; D3; endpoint facts P1.2; replica affinity planned | **P0** a "model switched" event and a portable conversation state | Service, Core | 1–2 |
| 14 | Budget, quota, cost | — (not verified on any platform) | `EngineLimits`, `sharedBudget`, `Usage`, `StopBudgetExhausted`; no cost | P1 cost field with `source` (reported/estimated/unknown) | Runtime, Core (Recipe budgets) | 2, 5 |
| 15 | Checkpoint / undo of file changes | Per-turn checkpoints outside git, restore code/conversation separately, limits stated (CC, OC, Cursor [S], Cline) | `effectLedger` keeps changed paths (`effects.go`); `port.ErrNoEffect`; **no pre-image** | **P0** an effect record with pre-image ref · P1 restore op | Runtime (Broker), Agent Folder | 3 |
| 16 | Hooks | Lifecycle hooks with allow/deny/ask/rewrite (CC, Cline [S], Cursor [S]) | `MemoryHook` Before/AfterTurn only | P1 hooks = Event subscribers + one authorize-time decision point. Its security rule is **P0** (see §4) | Runtime (Broker), Extensions | 4 |
| 17 | Skills | Progressive disclosure, scoped, can fork into a subagent (CC) | Skill packs, summaries in prompt (`service.go` `skillSummaries`), ceiling of 15 | P2 (scopes, fork) | Equipment | 4 |
| 18 | Permission modes | Named session modes; plan = read-only (CC, OC, Cline) | Approval classes auto/confirm/critical; `InteractiveAuthorizer` / `RejectingAuthorizer`; D1 floor | **P0** mode as a *narrowing* of the Policy Floor, recorded in events | Runtime (Broker) | 2 |
| 19 | Loop detection | Doom loop → ask (OC) | `MaxSteps`, `MaxRecoveries`, Broker duplicate by idempotency key | P1 a Recipe field (repeat threshold) | Core (Recipe), Runtime | 2 |

---

## 4. Adapt, don't copy: the P0 items in BOI's terms

1. **Identity (#1, #9).** Every event, checkpoint and effect carries
   `task_id`, `agent_id`, `parent_agent_id`, `step`, `call_id`, and a reserved
   `owner`/`lease_expires` in the checkpoint, empty until Phase 3+.
   *BOI style:* the ids are host-assigned, like `TaskSession`. The model never
   names its own id. SubAgent children get an `agent_id` from Runtime, not from
   the SubAgent Block.
2. **Event envelope v0 (#4, #6, #12).** Event v0 in ARCHITECTURE_IV §5.4 gains
   `schema_version`, a monotonic `seq` per task, `time`, the ids above,
   `recipe_version`, `kind`, and a versioned `payload`. Reserve these kinds now:
   `context.compacted`, `model.switched`, `approval.requested/decided`,
   `effect.recorded`, `mode.changed`, `capability.used{status}`.
   Migration rule: a reader accepts its own version and N-1, and unknown kinds
   are skipped rather than failing. *BOI style:* this is the Event contract with
   its own conformance test. It is not a new Block. The transport renders only
   events, which is the OpenCode lesson without copying its HTTP server.
3. **Transcript policy (#4).** Resume and fork need the conversation, but
   `TaskRecorder` currently forbids persisting prompts and model output. *BOI
   style:* keep the recorder rule, and add a separate opt-in transcript store
   in the Agent Folder, with redaction and retention. This changes what is kept
   on disk, so under plan rule 6 it needs **one owner question** before any
   code.
4. **Persisted approval (#3).** An `ApprovalRequest` is written to the Agent
   Folder as an event. It is bound to the call fingerprint, the
   `recipe_version` and the Provider. Resume re-validates all three: a changed
   Recipe or Provider invalidates it (D3, "no right grows"). An expired request
   is `ApprovalExpired`, never auto-approved (D2).
5. **Effect record with pre-image (#15).** Every verified change records
   `{call_id, path, pre_sha256, pre_ref, post_sha256}`. `pre_ref` points into
   the task's Agent Folder bin. Effects that cannot be reversed (`process.run`,
   network) are recorded as `reversible: false`. *BOI style:* undo is a
   capability with a status. `file edits: NATIVE undo`; `process.run:
   BLOCKED undo` (stated, never implied). This copies the honest limits list of
   Claude Code, not its UI.
6. **Typed content parts (#7).** `Message.Content` becomes `[]Part{Type, Text,
   MediaRef}`, with a `Text()` helper so existing adapters keep working.
   `vision` is a capability in the Gap Report with `UNVERIFIED` as the default.
   A text-only model with a captioning Equipment would be `ASSISTED`, never
   `NATIVE`. *BOI style:* Connector conformance fixtures reject a part type the
   endpoint facts do not declare. They never drop it silently.
7. **Model switch (#13).** Switching Provider mid-task emits `model.switched`,
   composes the new endpoint's Recipe, re-checks approvals and keeps the task
   state. Provider-specific state such as `reasoning_content` is marked
   non-portable and is dropped with a recorded reason. *BOI style:* this is D3
   applied to user-initiated switches, not only failover.
8. **Extension compatibility (#11).** `extension.yaml` declares
   `contracts: {connector: "0", event: "0", extension: "0"}`, a namespace
   (`<publisher>/<id>`), requested rights, and `depends_on` with version
   ranges. The loader refuses an Extension whose contract major does not
   match: it reports `BLOCKED: incompatible contract`.
9. **Permission mode as narrowing (#18, rule for #16).** A mode such as `plan`
   (read-only), `default` or `confirm-all` is a function that may only remove
   rights from the Policy Floor. The same rule binds hooks and Extensions: an
   authorize-time hook may return `deny` or `ask`, **never `allow` past the
   floor**, and cannot rewrite arguments after fingerprinting without creating
   a new call that is authorized again. Locking this now prevents a later
   "bypass" mode or hook from becoming a backdoor.

---

## 5. Experience Contract (draft)

Each guarantee holds whatever model, runtime or Extension is used.

| # | Guarantee | Delivered by | Test that proves it |
|---|---|---|---|
| E1 | **One instruction carries through.** The task runs until a named stop, and every stop states its reason and what the user can do next. | Runtime lifecycle, `StopReason` | Every engine exit path yields `StopReason.Valid()` and a non-empty notice, in a table test across the model zoo |
| E2 | **Progress is visible.** Every phase change, Tool call, approval and compaction appears as an event, and the screen shows only what events say. | Event contract, Transport | Replaying a recorded event log reproduces the TUI timeline; no step without an event |
| E3 | **Switching the model does not break the task.** The new model gets its own Recipe, approvals are checked again, and no right grows. | Core (Recipe), Service (Connector), D3 | Model-zoo test: switch from a native-tool endpoint to a dialect endpoint mid-task, then complete or stop with a stated reason |
| E4 | **A crash does not lose work or repeat it.** A killed task resumes from the Agent Folder without repeating a side effect. | Agent Folder checkpoints, Broker idempotency | The planned Crash Recovery and Tool Replay rows |
| E5 | **Changes can be undone, and what cannot be undone is said.** | Effect record with pre-image | Run a task, undo it, and file hashes equal the pre-images; a `process.run` effect is listed as not reversible |
| E6 | **Limits are honest.** Each capability shows its status, and `ASSISTED` is never shown as `NATIVE`. | Core (Gap Report), Event `capability.used` | Assert the rendered status equals the Recipe status for every capability in the zoo |
| E7 | **Nothing is approved silently.** Unattended runs refuse, approvals expire, and no hook, mode or Extension widens rights. | Broker, Policy Floor, D1/D2/D5 | Adversarial tests: hook returns `allow` on a floor-denied call, so the call is denied; non-interactive run, so `blocked` |
| E8 | **Cost and budget are visible.** Usage is shown with where its numbers came from, and running out is a named stop. | Runtime budgets, `Usage` | Budget test stops with `budget_exhausted`; cost shown as `unknown` when the Provider reports none |
| E9 | **The same guarantees hold in every front end.** CLI, TUI and API see the same events and stops. | Transport decides nothing | One scripted task run through each Transport yields identical event `kind` sequences |

---

## 6. Warnings: what would bloat Core if done speculatively

- **No agent-teams or messaging bus in Core.** Ids in events (P0) are enough.
  Messaging belongs to SubAgent/Runtime, after a real need is measured.
- **No scheduler, cron or daemon in Core or Runtime now.** Detach and cancel
  need an interface only. Scheduling is Transport or periphery.
- **No telemetry SDK (OpenTelemetry etc.) imported by Core.** Correlation ids
  are enough. Exporters are Extensions.
- **No vector DB or embedding store in memory now.** Add scope and provenance
  fields only. Retrieval engines are Equipment.
- **No LSP clients or IDE protocol in Runtime.** Diagnostics arrive as a Tool
  or Equipment feedback source. IDEs are Transport clients.
- **No multimodal adapters before a vision endpoint is in the model zoo.** Only
  the part type is fixed now.
- **No distributed lease or heartbeat before a Cluster has been observed.**
  ARCHITECTURE_IV §3 says no Cluster row is observed yet. Reserve the fields
  only.
- **Hooks that run arbitrary shell belong in Extensions (D5 approval).** They
  never live inside the Broker, and never have `allow` power.
- **Do not let the Event kind list grow without bound.** Use `kind` plus a
  versioned payload, with unknown kinds skipped, so new kinds never need a
  Core change.
- **Do not add a seventh Block** (for example "Session"). Sessions and
  transcripts belong to the Agent Folder, and identity to Runtime.
- **Keep the number of permission modes small (≤ 3).** Every mode needs its
  own enforcement tests under Rule 3.
- **Pricing tables are data in the Service catalogue, not Core logic.**
- **Skill scopes and forked skills are P2.** The 15/15 ceiling stays until
  evidence moves it.
