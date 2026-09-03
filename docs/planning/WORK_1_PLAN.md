# BOI Agent Suit Work 1 Plan

## Outcome

Deliver a production-usable single BOI Agent with one Core Persona, a user-named
Agent instance, Provider conformance checks, a bounded active registry of at
most 15 Skills and 15 Tools, one controlled Runtime, and predictable task
outputs. Work 1 does not include dynamic plugin installation, background Agents,
or SubAgent execution.

BOI Agent Suit is network-capable but is not an offline-first system. Local
components may exist, but disconnected operation is not a Work 1 guarantee.

## Current direction assessment

The current repository is aligned with Work 1 at the Runtime-foundation level,
but it is not yet a Work 1 product.

| Work 1 capability | Current evidence | Status | Readiness |
|---|---|---|---:|
| Six-Block ownership | Six manifests and conformance tests under `internal/block` | Complete foundation | 100% |
| One Core Persona | TUI and CLI use embedded Core Persona `boi`; switching is retired | Complete | 100% |
| User-named Agent | First run persists a versioned identity in `.boi/agent.yaml` | Complete | 100% |
| Provider conformance | Versioned deterministic probe suite, persisted profiles, and fail-closed Router composition | Complete implementation | 100% |
| Skill registry | Versioned explicit index, deterministic task selection, Context/dependency policy, and 15-active limit | Complete implementation | 100% |
| Tool registry | Versioned explicit index, Provider-gated active set, Broker enforcement, and 15-active limit | Complete implementation | 100% |
| Bounded Runtime | Engine, Broker, Approval, typed stops, budgets, cancellation, and tests exist | Strong foundation | 78% |
| Planning/verification/recovery | Ports and placeholder Planner/Reviewer exist; Service does not compose them | Partial | 35% |
| Agent Folder | No task output tray or manifest contract | Missing | 0% |
| Automation contract | `boi ask` exists; stable JSON, stream, exit-code, and no-TTY contracts are absent | Early foundation | 20% |
| Acceptance/evaluation | Unit and adversarial tests exist; no complete Work 1 task suite | Partial | 55% |

Weighted implementation readiness after W1.3 is approximately **74%**.
Architectural direction alignment is approximately **90%**, because the existing
Runtime safety model already matches Work 1 and SubAgents remain disabled.

## Scope boundaries

### Included

- Single Agent and single Core Persona `boi`.
- User-defined Agent name persisted independently from Persona.
- Provider capability flags based on reproducible conformance probes.
- Active Local Registry maximum of 15 Skills and 15 Tools.
- TUI and CLI composition through the same Core and Runtime.
- Task planning, authorization, execution, verification, and bounded recovery.
- Agent Folder with `bin`, `output`, evidence, and a versioned manifest.
- Read-only Automation and explicitly approved mutation paths.

### Excluded

- SubAgent execution.
- Dynamic Plugin or Skill installation.
- Background scheduling and autonomous long-running Agents.
- Multi-workspace execution in one task.
- A single subjective intelligence or reasoning score.
- Guaranteed disconnected operation.
- Unattended destructive or external side effects.

## Delivery sequence

### W1.0 - Architecture baseline

Status: **Complete** at commit `f5c0934`.

Tasks:

- Establish the six package boundaries.
- Declare ownership and non-ownership for every Block.
- Prevent concrete Blocks from importing one another directly.
- Preserve existing TUI and CLI behavior.

Acceptance:

- Exactly six valid and unique Block manifests.
- Cross-Block import conformance test passes.
- Full tests, vet, and build pass.

### W1.1 - Core identity

Status: **Complete**. Implemented and verified on 2026-09-02.

Tasks:

1. Define a versioned Agent Identity contract: ID, user-defined name, Core
   Persona reference, creation time, and schema version.
2. Make `boi` the only Core Persona used by Agent Service.
3. Replace first-run Persona selection with Agent naming.
4. Show Agent name in TUI status while keeping Provider/Model metadata separate.
5. Deprecate Persona switching without deleting legacy data before migration is
   verified.

Acceptance:

- A clean workspace asks for an Agent name exactly once.
- Restart loads the same identity.
- TUI and `boi ask` use Core Persona `boi`.
- Changing Provider does not change Agent identity.
- Legacy Persona commands have an explicit compatibility response.

Verification:

- Identity create/load and validation tests pass, including Thai Agent names.
- TUI status, splash, and `/persona` contract tests pass.
- Full repository tests, vet, build, and CLI compatibility smoke pass.

### W1.2 - Provider conformance

Status: **Complete implementation**. Live profiles are generated explicitly per
configured Provider/Model with `boi provider qualify <name>`.

Tasks:

1. Define a versioned Provider Capability Profile.
2. Implement deterministic probes for completion, Tool schema, Skill selection,
   Tool observation, authority compliance, and tested Context size.
3. Fingerprint profiles by Provider, Model, endpoint class, and probe version.
4. Separate `passed`, `failed`, `unsupported`, and `unverified` results.
5. Prevent failed capabilities from being exposed by environment composition.

Acceptance:

- Re-running unchanged probes produces an equivalent profile.
- A Provider failing Tool Calling receives no Tool Calling environment.
- Probe failure never grants a capability.
- Secrets and raw credentials never enter profiles or logs.

Verification:

- Deterministic fake-Provider probes cover completion, reasoning, BOI Tool
  Calling schema, Skill selection, untrusted Tool observations, authority, and
  tested Context payload.
- Repeated unchanged probes produce equivalent profiles while timestamps remain
  observational metadata.
- Router candidates without a valid passing completion profile are excluded.
- Tool and Skill environments are the fail-closed intersection of qualified
  Router candidates.
- Native Tool schema is explicitly `unsupported` until Provider transport
  carries native Tool definitions; it is never inferred from a model name.

### W1.3 - Active capability registries

Status: **Complete implementation**. Workspace indexes live under
`.boi/registry`; loose files are not runtime capabilities.

Tasks:

1. Define versioned Skill and Tool index entries.
2. Enforce maximum active counts of 15 Skills and 15 Tools.
3. Distinguish installed, eligible, selected, active, disabled, and rejected.
4. Select the working set using task need, Provider profile, required
   capabilities, Context budget, and policy.
5. Inject only selected Skill summaries; load full Skill instructions on use.
6. Keep all Tool execution behind Capability Broker.

Acceptance:

- The sixteenth active Skill or Tool is rejected deterministically.
- Unindexed files are never exposed automatically.
- A Skill cannot grant Tool authority.
- Selection is explainable and stable for the same inputs.

Verification:

- Both indexes require schema version, kind, unique names, source, summary, and
  explicit enabled policy.
- Selection decisions expose installed, eligible, selected, active, disabled,
  rejected, score, and reason fields.
- Tests prove deterministic rejection of the sixteenth Skill and Tool.
- Loose Skill files remain invisible until explicitly indexed.
- Broker rejects registered-but-inactive Tools, including MCP Tools.
- Agent Service receives only active Tool names and selected Skill summaries;
  full Skill instructions are retrievable only from the task's active set.

### W1.4 - Runtime composition

Tasks:

1. Add an explicit Task Plan contract with dependencies and status.
2. Compose Planner into Agent Service without exposing hidden authority.
3. Implement deterministic Tool Result verification.
4. Implement bounded recovery and re-planning policies.
5. Emit task, phase, Tool, approval, verification, and stop events.
6. Remove or isolate the legacy Agent Loop after parity tests pass.

Acceptance:

- Task tests demonstrate Plan -> Authorize -> Act -> Verify -> Re-plan/Complete.
- Verification cannot be satisfied by Model claims alone.
- Recovery respects retry, Tool, time, token, and step budgets.
- TUI and CLI use the same composition root and policy.

### W1.5 - Agent Folder

Tasks:

1. Define one Agent Folder root with only `bin` and `output` as primary trays.
2. Create task-scoped directories and stable Task IDs.
3. Route temporary, log, draft, checkpoint, failed, and recovery material to
   `bin/<task-id>`.
4. Route user deliverables to `output/<task-id>`.
5. Write a versioned manifest containing outcome, artifacts, evidence,
   timestamps, Provider profile reference, and final Stop Reason.
6. Define retention and cleanup without deleting user deliverables by default.

Acceptance:

- Every completed task has one discoverable manifest.
- The manifest identifies every deliverable without repository-wide search.
- Temporary material never appears as a completed deliverable.
- Logs and manifests contain no secrets.

### W1.6 - Automation contract

Tasks:

1. Finalize argv/stdin inputs and stdout/stderr separation.
2. Add versioned `--json` result output.
3. Define stable exit codes for completed, invalid input, denied, cancelled,
   unavailable, verification failure, and internal error.
4. Define no-TTY behavior and non-interactive approval outcomes.
5. Add idempotency keys and artifact references.
6. Enable read-only Automation first; mutation remains explicitly authorized.

Acceptance:

- A script can parse results without reading human TUI text.
- Repeating an idempotent request does not duplicate side effects.
- Non-interactive execution never hangs waiting for approval.
- Mutation without a valid authorization contract does not execute.

### W1.7 - Product acceptance

Tasks:

1. Add end-to-end task fixtures covering explanation, repository inspection,
   report creation, approved write, rejection, cancellation, Provider failure,
   and recovery.
2. Add compatibility tests for migrated workspaces.
3. Update README and command reference to remove six-Persona and simulated-mode
   claims that no longer match the product.
4. Run test, vet, build, CLI smoke, TUI smoke, and task-level evaluation.
5. Produce release and rollback notes.

Acceptance:

- All Work 1 gates pass from a clean workspace and a migrated workspace.
- No Work 2 feature is required for normal Work 1 operation.
- SubAgent remains disabled.
- Observed behavior matches the published contracts.

## Dependency map

```text
W1.0 Architecture baseline [complete]
  -> W1.1 Core identity [complete]
      -> W1.2 Provider conformance [complete]
          -> W1.3 Active registries [complete]
              -> W1.4 Runtime composition [next]
                  -> W1.5 Agent Folder
                      -> W1.6 Automation contract
                          -> W1.7 Product acceptance
```

W1.5 schema design may begin while W1.4 is being implemented, but Runtime must
own the final Task Result contract. W1.6 must consume the Agent Folder manifest
rather than inventing a second result format.

## Work 1 success target

The target probability remains **90-92%** only if the exclusions remain firm,
the migration is incremental, and each phase closes its acceptance gate before
the next phase changes runtime behavior.
