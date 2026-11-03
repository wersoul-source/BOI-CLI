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

**Read order:** ADR-006 → `ARCHITECTURE_IV.md` → this file →
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

### Phase 1 — Connector and protocol

| ID | Task |
|---|---|
| P1.1 | Split the OpenAI adapter into Transport Adapter, Protocol Adapter and Dialect Parser behind the existing interface |
| P1.2 | Endpoint facts reported to Core (streams, native tools, parameter names, reasoning replay) |
| P1.3 | Dialect parsers: DSML, Hermes, Qwen XML (the 2026-10-24 DSML reply is the first fixture) |
| P1.4 | Carry and replay `reasoning_content` |
| P1.5 | Record every fallback with its reason; `rejectsNativeReplay` no longer downgrades silently |

### Phase 2 — Calibration Core

| ID | Task |
|---|---|
| P2.1 | `Recipe` type and `ComposeRecipe(profile, endpoint facts)` in Core, table-tested |
| P2.2 | Four capability statuses carried from the Gap Report into the Recipe |
| P2.3 | Approval Floor in the authorisation path (D1) with refusal in non-interactive runs (D2); enforcement tests make D1 routes `ASSISTED` |
| P2.4 | Recipe per endpoint and a shared Policy Floor in the Router (D3) |
| P2.5 | Drive the Service switches from the Recipe; keep exclusion only for no profile and failed completion |
| P2.6 | Text, planning and analysis stay available when Tools are `UNVERIFIED` or `BLOCKED`, with the limitation stated |

### Phase 3 — Recoverable Runtime

Checkpoint and resume through the Agent Folder; no side-effecting Tool replays
without approval or a proven idempotency guarantee; compaction by summary;
budgets from the Recipe; circuit breaker and replica affinity.

### Phase 4 — Composable Extensions

`extension.yaml` (extends `PackManifest`), validator, Extension Registry and
SDK; approval per D5; Extension Isolation tests; OS-level sandbox on Linux for
`process.run` and code Extensions (D7); Agent-built Extensions.

### Phase 5 — Measure and adapt

`codingbench` per source and trait, before/after each compensation on
correctness, safety, stability and cost; Recipe versions, shadow runs, promotion
and rollback; Provider drift detection.

### Phase 6 — Migrate and release

Config migration and a v2 → v3 guide; real acceptance on the D6 model set;
contracts frozen; rollback path documented.

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
| P0.6 | review | `internal/acceptance/modelzoo` + `modelzoo_test.go`; baseline: 5 traits pass, 5 gaps recorded (DSML, Hermes, Qwen XML → P1.3; reasoning replay → P1.4/P1.5; corrupt token → P2.3); dead replica fails over |
| P0.7 | todo | needs a recorded real run from the owner |
| Phase 1 | todo | next; the zoo rows for DSML, Hermes, Qwen XML and reasoning replay are its acceptance |
| Phase 2 | todo | D1–D3 approved; needs P0.6 fixtures |
| Phases 3–6 | todo | |

---

## 4. Acceptance matrix additions

Recorded in
[`ACCEPTANCE_TEST_MATRIX.md`](../architecture/composable/ACCEPTANCE_TEST_MATRIX.md)
as planned rows: every completing Provider gets a Recipe; Host security
invariants; Tool Replay; Recipe Rollback; Provider Drift; Extension Isolation;
Crash Recovery.

"100%" for any of these means every test defined for it passes. It does not
mean the software has no bugs or supports every model perfectly.
