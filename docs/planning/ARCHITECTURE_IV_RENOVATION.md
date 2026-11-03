# Architecture IV — Renovation plan

Status: **active**. Approved by the owner on 2026-11-02 with
[ADR-006](../architecture/ADR-006-calibration.md) and decisions D1–D7.
Design: [`ARCHITECTURE_IV.md`](../architecture/ARCHITECTURE_IV.md).
Target: v3.0.0.

This plan supersedes [`RENOVATION_PLAN.md`](RENOVATION_PLAN.md) as the plan of
record. That file stays as the evidence and history of R0–R5; its tasks are
folded into the phases below.

---

## 0. Rules for the agent picking this up

**Read order:** ADR-006 → `ARCHITECTURE_IV.md` (§5.5 foundation amendments,
§6 Experience Contract) → this file →
[`HANDOFF.md`](../../HANDOFF.md) →
[`docs/audit/2026-11-01-full-audit.md`](../audit/2026-11-01-full-audit.md).

1. Inspect the code, contracts, tests and the live state of `master` before
   changing anything. This file records a plan, not immutable truth.
2. One task, one revertable commit; group commits of one phase into one PR when
   a single branch is all you have.
3. Every PR answers **Gap / Composition / Evidence** (ADR-006,
   `.github/PULL_REQUEST_TEMPLATE.md`), or `N/A` with a reason.
4. Use the four statuses exactly: `NATIVE`, `ASSISTED`, `UNVERIFIED`,
   `BLOCKED`. Never report an `ASSISTED` capability as `NATIVE`, and never
   claim a model or the system works without a test result.
5. Keep the Trusted Foundation intact (ARCHITECTURE_IV §2). No compensation may
   bypass a safety rule.
6. Do not start work that changes safety policy beyond what D1–D7 settle;
   ask the owner one precise question instead.
7. Never put an API key or token in chat, code, docs, commits or fixtures.
   Scripts read secrets with `read -s`.
8. Cloud sessions cannot push tags (HTTP 403) and cannot reach most Provider
   hosts. Release tags and real-Provider runs are done by the owner, with the
   exact commands supplied.
9. Gates before every push, each step confirming itself:
   `test -z "$(gofmt -l .)"`, `go vet ./...`, `staticcheck ./...`,
   `go test -race -count=1 ./...`, `go build ./cmd/boi`.
10. Update the status board (§3) in the same PR as the work.
11. Rule 4 (ADR-006): before building a feature, check whether it touches a
    Foundation amendment F1–F9 (ARCHITECTURE_IV §5.5). Build to that shape;
    never invent a parallel one. Ideas from other platforms come in as
    principles restated in BOI's terms, never as copies.
12. Experience acceptance: a PR that changes what the user sees names which
    Experience Contract rows (E1–E9) it affects and how they were checked.

---

## 1. Phases

Phases 1 and 2 may overlap in development. Turning on the new Tool policy needs
Phase 2 tests to pass first. Phase 4 must not let an Extension replace any part
of the Trusted Foundation.

| Phase | Goal | Folds in | Done when |
|---|---|---|---|
| **0 · Contracts and foundation** | Principle, contracts and test fixtures fixed before the Runtime changes | R0 | ADR and contracts merged, records consistent, and CI runs one task set against a simulated model of every kind |
| **1 · Connector and protocol** | Talk to every model source through Service | R2 | Simulated endpoints and gateways of every kind complete a Tool call through the Connector fixtures |
| **2 · Calibration Core** | Gap Report → Recipe → composition replaces the on/off switches | R1 | Every Provider that completes gets a Recipe; Maxplus regains Tools as `ASSISTED` with every call approved |
| **3 · Recoverable Runtime** | Long tasks survive failure | R3 | A task that dies midway resumes without repeating a side effect |
| **4 · Composable Extensions** | Users and the Agent add capabilities without touching Core | roadmap: skill authoring, MCP connect, plugin layer | Install and remove an Extension without changing Core; Extension Isolation tests pass; OS sandbox on Linux (D7) |
| **5 · Measure and adapt** | Prove which compensations help | R4 | Before/after numbers per compensation on both axes; ones that do not help are removed; Recipe shadow and rollback work |
| **6 · Migrate and release v3.0.0** | Ship | — | Real acceptance on Windows, Linux and Android against Local, API, Gateway and Cluster; contracts frozen; no task reported done without evidence |

---

## 2. Phase tasks

### Phase 0 — Contracts and foundation

| ID | Task | Touch points | Acceptance |
|---|---|---|---|
| P0.1 | ADR-006: principle, three rules, four statuses, D1–D7, PR criteria | `docs/architecture/ADR-006-calibration.md` | merged |
| P0.2 | Architecture IV design and the four contracts at v0 | `docs/architecture/ARCHITECTURE_IV.md` | merged |
| P0.3 | Reword every record that still describes a gate as the principle; keep current-behaviour notes accurate | `architecture-6-blocks.md`, `BLOCK_ARCHITECTURE.md`, `BOI_V17_GAP_ANALYSIS.md`, `ACCEPTANCE_TEST_MATRIX.md`, `WORK_1_PLAN.md`, `current-state.md`, `RENOVATION_PLAN.md`, `HANDOFF.md` | no document presents "failed probe → Tools off" as the design |
| P0.4 | PR template asks Gap / Composition / Evidence | `.github/PULL_REQUEST_TEMPLATE.md` | merged |
| P0.5 | Capability Gap Report v0: `boi env` explains every configured Provider (profile state, gaps with evidence); `qualify` prints the reason for every non-passed result | `internal/app/report.go`, `internal/transport/cli/env.go`, `provider.go` | `TestReportExplainsEveryProviderGap`, `TestEnvPrintsEachProviderGap`, `TestQualifyPrintsTheReasonForEveryResult` |
| P0.6 | **Simulated model zoo**: `httptest` endpoints that reproduce each trait and source seen or expected — no streaming, DSML tool calls, Hermes and Qwen XML, `reasoning_content` required, tool calls without `index`, a corrupted token, slow first load, a failing replica | new test package under `internal/` | one scripted task runs against every simulated endpoint in CI, and each records its expected status |
| P0.7 | Record and replay real conversations so a bug seen with a real Provider becomes a deterministic test | test helper + fixture format | a recorded maxplus probe run replays in CI |
| P0.8 | **Foundation amendments (Rule 4)**: lock the shapes F1–F9 and the Experience Contract E1–E9, from a shallow survey of five platforms | `ADR-006` Rule 4, `ARCHITECTURE_IV.md` §5.5 and §6, `docs/research/2026-11-03-platform-principles.md` | documents merged; each F item named in the phase that builds it; F9 has an owner decision |

### Phase 1 — Connector and protocol

| ID | Task |
|---|---|
| P1.1 | Split the OpenAI adapter into Transport Adapter, Protocol Adapter and Dialect Parser behind the existing interface |
| P1.2 | Endpoint facts reported to Core (streams, native tools, parameter names, reasoning replay) |
| P1.3 | Dialect parsers: DSML, Hermes, Qwen XML (the 2026-10-24 DSML reply is the first fixture) |
| P1.4 | Carry and replay `reasoning_content` |
| P1.5 | Record every fallback with its reason; `rejectsNativeReplay` no longer downgrades silently |
| P1.6 | Typed content parts on `Message` (F3), text only; endpoint facts declare accepted part types |
| P1.7 | Event envelope v0 (F2): `schema_version`, `seq`, ids (F1), `recipe_version`; N-1 reader and unknown-kind skip, with a conformance test |

### Phase 2 — Calibration Core

| ID | Task |
|---|---|
| P2.1 | `Recipe` type and `ComposeRecipe(profile, endpoint facts)` in Core, table-tested |
| P2.2 | Four capability statuses carried from the Gap Report into the Recipe |
| P2.3 | Approval Floor in the authorisation path (D1) with refusal in non-interactive runs (D2); enforcement tests make D1 routes `ASSISTED` |
| P2.4 | Recipe per endpoint and a shared Policy Floor in the Router (D3) |
| P2.5 | Drive the Service switches from the Recipe; keep exclusion only for no profile and failed completion |
| P2.6 | Text, planning and analysis stay available when Tools are `UNVERIFIED` or `BLOCKED`, with the limitation stated |
| P2.7 | Model switch (F4) by failover or by the user: `model.switched`, new Recipe, approvals re-checked, non-portable state dropped with a reason; zoo test for E3 |
| P2.8 | Permission modes as narrowing only (F8): `plan`, `default`, `confirm-all`, with enforcement tests |

### Phase 3 — Recoverable Runtime

Checkpoint and resume through the Agent Folder; no side-effecting Tool replays
without approval or a proven idempotency guarantee. Built to the Foundation
shapes F1, F5, F6, F9. Task breakdown proposed 2026-11-03 (re-check against
the code before starting).

| ID | Task |
|---|---|
| P3.1 | Owner decision on the transcript store (F9), then the opt-in store with redaction and retention; resume and fork need it |
| P3.2 | Checkpoint carries the F1 ids and the reserved `owner` / `lease_expires`; resume reads checkpoint plus event log |
| P3.3 | Persisted approval (F5): `approval.requested` / `approval.decided` bound to call fingerprint, Recipe version and Provider; resume re-checks all three |
| P3.4 | Effect record with pre-image (F6) and undo for file edits; `process.run` and network effects listed as not reversible |
| P3.5 | Tool Replay guard: a resumed task never repeats a side effect without approval or a proven idempotency key |
| P3.6 | Context compaction by summary with a `context.compacted` event; budgets taken from the Recipe |
| P3.7 | Circuit breaker and replica affinity in the Router |
| P3.8 | Crash Recovery acceptance: kill a task mid-way, resume, no repeated side effect (E4) |

### Phase 4 — Composable Extensions

`extension.yaml` extends `PackManifest`. Built to F7 and F8, D5 and D7. No
Extension may replace any part of the Trusted Foundation.

| ID | Task |
|---|---|
| P4.1 | `extension.yaml` schema and validator: contract majors, namespace `<publisher>/<id>`, requested rights, `depends_on` ranges (F7) |
| P4.2 | Extension Registry: install, enable, disable, remove; dependency resolution; a contract mismatch reports `BLOCKED: incompatible contract` |
| P4.3 | Approval per D5: before install and first enable, and again when rights or trusted code change |
| P4.4 | Extension Isolation tests: no secrets, no widened workspace scope, Trusted Foundation untouchable |
| P4.5 | Hooks as Event subscribers plus one authorisation point that can answer only `deny` or `ask` (F8) |
| P4.6 | `SandboxBackend` interface that reports its guarantees, and the Linux OS-level implementation for `process.run` and code Extensions (D7) |
| P4.7 | Extension SDK; skill authoring and MCP connect delivered as Extensions |
| P4.8 | Agent-built Extensions: generate, manifest, test, approval, install |

### Phase 5 — Measure and adapt

| ID | Task |
|---|---|
| P5.1 | `codingbench` per source and trait: one task set, before and after each compensation |
| P5.2 | Numbers on correctness, safety, stability and resource cost per compensation, against criteria set in advance (Rule 3) |
| P5.3 | Recipe versions, shadow runs, promotion and rollback |
| P5.4 | Provider drift detection: re-probe when behaviour or fingerprint changes |
| P5.5 | Remove every compensation that does not help |

### Phase 6 — Migrate and release

| ID | Task |
|---|---|
| P6.1 | Config migration and the v2 → v3 guide |
| P6.2 | Real acceptance on the D6 set (small local model, direct API, Maxplus, DeepSeek direct) on Windows, Linux and Android |
| P6.3 | Freeze the four contracts (Connector, Recipe, Extension, Event) |
| P6.4 | Documented rollback path; release v3.0.0 (the owner pushes the tag) |

---

## 3. Status board

`todo` · `doing` · `review` · `done` · `blocked`

| ID | Status | Note |
|---|---|---|
| P0.1 | done | PR #26 |
| P0.2 | done | PR #26 |
| P0.3 | done | PR #26 |
| P0.4 | done | PR #26 |
| P0.5 | done | PR #26 |
| P0.6 | done | PR #27. Its DSML fixture had a format bug (parameter name held the tool name); fixed with P1.3 |
| P0.7 | todo | needs a recorded real run from the owner |
| P0.8 | done | PR #28; F9 (transcript store) waits for the owner, now P3.1 |
| P1.3 | done | PR #28 |
| P1.4, P1.5 | done | PR #29 |
| P1.7 | review | this PR: `internal/runtime/agent/event.go` (envelope, kinds, N-1 reader), `events.jsonl` per task in the Agent Folder, metadata only; `route.changed` added to the kinds. SubAgent runs are not recorded yet (they have no task session), so `parent_agent_id` is wired but unexercised until P3.2 |
| P1.1, P1.2, P1.6 | todo | P1.1/P1.2 split the adapter and report endpoint facts; P1.6 typed content parts |
| Phase 2 (P2.1–P2.8) | todo | D1–D3 approved; zoo fixtures ready (corrupt-token row is P2.3) |
| Phase 3 (P3.1–P3.8) | todo | P3.1 needs the owner's decision first |
| Phases 4–6 (P4.1–P6.4) | todo | |

---

## 4. Acceptance matrix additions

Recorded in
[`ACCEPTANCE_TEST_MATRIX.md`](../architecture/composable/ACCEPTANCE_TEST_MATRIX.md)
as planned rows: every completing Provider gets a Recipe; Host security
invariants; Tool Replay; Recipe Rollback; Provider Drift; Extension Isolation;
Crash Recovery.

"100%" for any of these means every test defined for it passes. It does not
mean the software has no bugs or supports every model perfectly.
