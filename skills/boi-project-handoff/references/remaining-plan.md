# Remaining Plan

Do not treat this plan as permission to implement every phase at once. Complete one acceptance gate, update the handoff, then continue.

The authoritative plan for the 90% target is
[`docs/planning/WORK_1_PLAN.md`](../../../docs/planning/WORK_1_PLAN.md). The
historical H-phase mapping below remains only for traceability.

## Phase H0 - Handoff and mobile baseline

- Publish the handoff Skill and knowledge package.
- Add Android ARM64 to release build targets.
- Clone/build/run on the physical S25+.
- Record TUI, CLI, filesystem, cancellation, Provider, and service-availability results.

Acceptance: the exact commit runs on S25+, `boi version`, `boi doctor`, one TUI query, and one `boi ask` query are observed. No secrets enter Git.

Repository handoff, Android build target, and cross-build are complete. Physical
S25+ acceptance is owned by the separate mobile path and does not block H1-H5.

## Phase H1 - Six-Block skeleton

- Introduce package boundaries without changing behavior.
- Define Block interfaces and dependency direction.
- Add architecture conformance tests.

Acceptance: no circular ownership, existing tests pass, and TUI/CLI behavior remains compatible.

Status: complete. Six manifests and direct-cross-Block import conformance tests
exist under `internal/block`. No existing runtime behavior was moved.

## Phase H2 - Core identity and qualification

- Make `boi` the only Core Persona. **Complete in W1.1.**
- Add first-run Agent naming and persisted instance identity. **Complete in W1.1.**
- Add Provider tests for Tool Calling, Skill Calling, reasoning, Context, authority, and verification behavior.
- Produce a versioned Capability Profile.

Acceptance: the same Provider profile is reproducible; failed capabilities reduce the environment rather than being silently assumed.

Status: complete through W1.2. Profiles are generated explicitly with
`boi provider qualify <name>`; unqualified Providers do not enter Agent routing.

## Phase H3 - Service and capability registries

- Add Local Skill and Tool indexes. **Complete in W1.3.**
- Enforce active maximums of 15 each. **Complete in W1.3.**
- Add dependency reporting, Provider/MCP health, MCP discovery, and Library routing.

Acceptance: unavailable required services produce a typed unavailable state; registry selection is deterministic and explainable.

Status: Local Registry and selection are complete. Service health, dependency
reporting, MCP discovery, and Library routing remain outside W1.3.

## Phase H4 - Runtime composition

- Compose Planner, real Verifier, bounded Recoverer, Skill calls, Tool calls, Context policy, and status events. **Complete for Work 1.**
- Preserve host-owned authority and typed stops. **Complete for Work 1.**

Acceptance: task-level evaluations prove Plan -> Act -> Verify -> Re-plan and no Tool bypass.

Status: complete in W1.4. Legacy direct execution paths were removed; Broker is
the sole Tool execution path. Continue with Agent Folder.

## Phase H5 - Agent Folder

- Route temp/checkpoint/log material to `bin`.
- Route deliverables to per-task `output` directories.
- Add versioned artifact manifests and cleanup policy.

Acceptance: a user can find every deliverable from one manifest without repository-wide search.

Status: complete in W1.5. TUI and CLI compose the same workspace-level Agent
Folder; completed manifests live under output, diagnostic state lives under
bin, and cleanup cannot automatically delete output.

## Phase H6 - Automation contract

- Finalize stdin/argv contracts, stdout/stderr separation, exit codes, `--json`, no-TTY behavior, idempotency, and non-interactive approval policy.
- Add Automation task/result schemas and artifact references.

Acceptance: read-only Automation is deterministic; side effects cannot run without an explicit machine-safe authorization contract.

Status: complete for Work 1 read-only Automation. JSON schema v1, argv/stdin,
streams, exits, cancellation, artifact references, and hashed idempotency keys
are implemented. Mutation remains denied and cross-process replay is not
claimed.

## Phase H7 - SubAgent gate

- Implement SubAgent Skill, Index, packages, delegation contracts, isolated budgets, and evaluation.
- Keep disabled until owner acceptance.

Acceptance: a child cannot exceed delegated Context, Tool, Workspace, time, token, or side-effect authority.
