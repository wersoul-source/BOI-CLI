# Remaining Plan

Do not treat this plan as permission to implement every phase at once. Complete one acceptance gate, update the handoff, then continue.

## Phase H0 - Handoff and mobile baseline

- Publish the handoff Skill and knowledge package.
- Add Android ARM64 to release build targets.
- Clone/build/run on the physical S25+.
- Record TUI, CLI, filesystem, cancellation, Provider, and offline results.

Acceptance: the exact commit runs on S25+, `boi version`, `boi doctor`, one TUI query, and one `boi ask` query are observed. No secrets enter Git.

## Phase H1 - Six-Block skeleton

- Introduce package boundaries without changing behavior.
- Define Block interfaces and dependency direction.
- Add architecture conformance tests.

Acceptance: no circular ownership, existing tests pass, and TUI/CLI behavior remains compatible.

## Phase H2 - Core identity and qualification

- Make `boi` the only Core Persona.
- Add first-run Agent naming and persisted instance identity.
- Add Provider tests for Tool Calling, Skill Calling, reasoning, Context, authority, and verification behavior.
- Produce a versioned Capability Profile.

Acceptance: the same Provider profile is reproducible; failed capabilities reduce the environment rather than being silently assumed.

## Phase H3 - Service and capability registries

- Add Local Skill and Tool indexes.
- Enforce active maximums of 15 each.
- Add Foundation bundle, dependency resolution, offline mode, MCP discovery, and Library routing.

Acceptance: clean offline startup works without downloading; registry selection is deterministic and explainable.

## Phase H4 - Runtime composition

- Compose Planner, real Verifier, bounded Recoverer, Skill calls, Tool calls, Context policy, and status events.
- Preserve host-owned authority and typed stops.

Acceptance: task-level evaluations prove Plan -> Act -> Verify -> Re-plan and no Tool bypass.

## Phase H5 - Agent Folder

- Route temp/checkpoint/log material to `bin`.
- Route deliverables to per-task `output` directories.
- Add versioned artifact manifests and cleanup policy.

Acceptance: a user can find every deliverable from one manifest without repository-wide search.

## Phase H6 - Automation contract

- Finalize stdin/argv contracts, stdout/stderr separation, exit codes, `--json`, no-TTY behavior, idempotency, and non-interactive approval policy.
- Add Automation task/result schemas and artifact references.

Acceptance: read-only Automation is deterministic; side effects cannot run without an explicit machine-safe authorization contract.

## Phase H7 - SubAgent gate

- Implement SubAgent Skill, Index, packages, delegation contracts, isolated budgets, and evaluation.
- Keep disabled until owner acceptance.

Acceptance: a child cannot exceed delegated Context, Tool, Workspace, time, token, or side-effect authority.
