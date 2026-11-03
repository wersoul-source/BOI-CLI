# ADR-006: Calibrate the environment to the model

Status: **accepted** by the owner on 2026-11-02, together with decisions D1–D7.
Scope: the whole of Architecture IV ([`ARCHITECTURE_IV.md`](ARCHITECTURE_IV.md)).
Plan: [`docs/planning/ARCHITECTURE_IV_RENOVATION.md`](../planning/ARCHITECTURE_IV_RENOVATION.md).

The owner's draft of this decision was titled "ADR-014". The repository's
sequence continues at 006; it is the same decision.

## Principle

> BOI ต้องปรับสภาพแวดล้อมให้เหมาะกับโมเดล ไม่ใช่บังคับให้โมเดลเหมาะกับ BOI
>
> BOI adapts its environment to the model. It does not force the model to fit BOI.

With one condition that overrides every compensation: **adding a capability
must never become pretending that the model can do what it cannot.**

## Context

Qualification was built as a gate. `ComposeEnvironment` could only switch
capabilities off, the acceptance matrix recorded "failed Tool-calling probe →
completion-only Agent" as implemented, and Core's own definition listed a
"qualification gate". The word *calibration* appeared in that definition, but
no code calibrated anything.

The cost showed on a real run (2026-10-29, `maxplus`/`deepseek-v4.1-flash`):
the endpoint passed native tool calling, yet one corrupted token in the
`tool_observation` probe (`OBSERVATION_public` for `OBSERVATION_IGNORED`)
withheld every Tool. The model never followed the injected instruction; the
gate could not tell a safety failure from a fidelity slip.

## Decision

### Rule 1 — Fill, don't cut

Core finds the gap between what a task needs and what the model can actually
do, and records it in a **Capability Gap Report**. Every capability that did
not pass states its cause, its evidence, and the compensations that might close
it: an Adapter, a Tool, a prompt Recipe, or a Runtime configuration.

A failed probe does not switch a capability off by itself. It also does not
switch it on: a capability is enabled only once it is proven safe and working.

### Rule 2 — Small core, detachable periphery

Core inspects, chooses components, checks policy and confirms results. It does
not hold fixes for a particular Provider or model. Runtime installs and
configures; Equipment provides the attachable capabilities (Tool adapters,
Memory, Context Manager, a vision bridge).

Every component has an interface, a version, its dependencies and a way to be
removed without damaging Core.

### Rule 3 — Measure before trusting

Every compensation is tested before and after, on the same task, under
comparable conditions, on at least four dimensions: **correctness, safety,
stability and resource cost**. It is enabled only if it meets criteria set in
advance. Without evidence it stays `UNVERIFIED`. Every confirmed bug gets a
regression test.

### Capability status

| Status | Meaning | BOI's behaviour |
|---|---|---|
| `NATIVE` | The model does it alone and passed the test | Enabled |
| `ASSISTED` | The model does it with a compensation, and passed the test with it | Enabled, and reported as assisted |
| `UNVERIFIED` | Not enough evidence yet | Not used in real work |
| `BLOCKED` | A safety risk, or no compensation meets the criteria | Not used; the risky path only, with the evidence kept for further work |

Prohibited:

- An Adapter or any compensation that bypasses a safety rule.
- Reporting a capability provided by Equipment as the model's own. An
  `ASSISTED` capability is never shown as `NATIVE`.

Not every limit can be closed by adding a component. A limit proven to be the
model's own stays reported as such.

### Exclusion

Exactly two cases keep a Provider out of the Router:

- **No measured profile** (never qualified, expired, or unreadable): its
  capabilities are `UNVERIFIED`. This keeps the existing acceptance test
  "unqualified Provider fails before task creation".
- **Completion does not work**: there is nothing to compensate. It is
  `BLOCKED` until the connection is repaired or the endpoint is changed.

A model that completes but cannot be made to call Tools is still used to
answer, plan and analyse, and its Tool limitation is stated plainly.

### Decisions D1–D7

| | Decision |
|---|---|
| D1 | When a safety probe (`tool_observation`, `authority`) fails, the Tool route stays open but every call needs approval through the Broker. Approval does not replace the sandbox or policy, because the person approving may not see every side effect of a command. |
| D2 | In a non-interactive run (`boi ask`, automation), a call that needs approval is refused and recorded as blocked with its reason. Nothing is approved automatically. |
| D3 | With several Providers in the Router: a shared, strict Policy Floor, and a Recipe per endpoint from the first implementation. A failover uses the new Provider's Recipe and checks approval again. No Provider borrows another's probe results. No right grows from a retry or a Provider switch. |
| D4 | Architecture IV is the target of **v3.0.0** (major). It is a target, not a released version. |
| D5 | An Extension the Agent builds needs approval before it is installed and first enabled, and again whenever it asks for more rights or its trusted code changes. Re-enabling it under the same rights does not. |
| D6 | Release testing uses a small local model, a direct API, Maxplus and DeepSeek direct, plus fixtures for a gateway that does not stream and for a cluster. |
| D7 | `process.run` has a path boundary and a best-effort command deny-list, not OS-level isolation. v3.0.0 relies on the path boundary, approval of every call and secret isolation; OS-level sandboxing is built in Phase 4, Linux first, because Extension isolation needs it anyway. |

#### How D1 meets Rule 3

The approval floor of D1 is a compensation, so it obeys Rule 3 like any other.
A Tool route under D1 is `ASSISTED` only once tests prove that every call on it
reaches approval, that a non-interactive run refuses it (D2), and that no
Provider switch or retry escapes it (D3). Until those tests pass, the route is
`UNVERIFIED` and stays closed.

### Pull request acceptance

From 2026-11-02 every PR answers three questions in its description:

1. **Gap** — what limitation was found, and what is the evidence?
2. **Composition** — what is added, which Block owns it, and how is it removed?
3. **Evidence** — how do before and after differ, which criteria did it meet,
   and is there a regression test?

A PR that does not change what a model can do (documentation, tooling) answers
`N/A` with a reason. `.github/PULL_REQUEST_TEMPLATE.md` carries the questions.

### Worked example: Maxplus

Old: one failed probe → every Tool withheld.

New:

1. Find whether the failure comes from the model, the prompt, the Tool schema
   or the Provider (the Gap Report already shows the model's actual reply).
2. Try the relevant Recipe or Adapter in isolation.
3. Test again, both the normal cases and the one that failed.
4. If it meets the criteria, enable that capability as `ASSISTED`.
5. If not, `BLOCKED` the risky path only, and keep the evidence.

## Consequences

- Composition stops being subtractive. Profile → Recipe → Runtime replaces the
  boolean `AgentEnvironment` switches in Phase 2; until then today's behaviour
  stands and is documented as current, not as the principle.
- Every capability carries one of four states, so the product can say "works
  because BOI helps" honestly instead of "works" or "off".
- More tests: every compensation needs its before/after measurement and every
  safety compensation its enforcement tests. That is the intended cost.
- The Trusted Foundation (Broker, permission enforcement, secret isolation,
  sandbox and workspace boundaries, evidence ledger, schema validation,
  recovery limits) cannot be replaced by a plug-in under any status.
- Superseded wording: the "qualification gate" in Core's definition, the
  "Failed Tool-calling probe → completion-only Agent" row of the v1.7 gap
  analysis, and row 6 of the acceptance matrix are reworded to match this ADR.
