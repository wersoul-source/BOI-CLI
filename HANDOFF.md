# BOI CLI Handoff

This is the entry point for Kampun or any Agent continuing BOI CLI.

## Repository

- Remote: `https://github.com/wersoul-source/BOI-CLI.git`
- Primary branch: `master`
- Historical architecture baseline: `f5c0934`
- Pre-W1.7 rollback baseline: `f3db72b`
- Product version in source: `0.3.0`
- Language: Go `1.24.2` or later compatible toolchain
- Main interfaces: TUI (`boi`) and non-interactive CLI (`boi ask`)

Always inspect the live commit and working tree; this file records a handoff baseline, not immutable truth.

## Read in this order

1. [`skills/boi-project-handoff/SKILL.md`](skills/boi-project-handoff/SKILL.md)
2. [`current-state.md`](skills/boi-project-handoff/references/current-state.md)
3. [`architecture-6-blocks.md`](skills/boi-project-handoff/references/architecture-6-blocks.md)
4. [`remaining-plan.md`](skills/boi-project-handoff/references/remaining-plan.md)
5. [`s25-plus.md`](skills/boi-project-handoff/references/s25-plus.md)
6. [`WORK_1_PLAN.md`](docs/planning/WORK_1_PLAN.md)

## Owner-approved direction

BOI Agent Suit has six Blocks: Service, Core, Various Equipment, Runtime, Agent Folder, and SubAgent. It has one Core Persona, `boi`; the user names their Agent instance during first TUI/GUI onboarding. Core evaluates each Provider Model and composes an environment from a bounded active registry of at most 15 Skills and 15 Tools. BOI Agent Suit is network-capable but is not an offline-first system.

## Current continuation gate

Work 1 Phase W1.0 introduced the six package boundaries and conformance tests.
W1.1 fixes the runtime Persona to embedded `boi`, persists a separately
named Agent identity in `.boi/agent.yaml`, displays that name in TUI, and keeps
legacy Persona commands as non-destructive compatibility paths. W1.2 adds
versioned Provider probes and profiles; unqualified Provider candidates cannot
enter the Agent Router, and Tool/Skill environments compose fail-closed.
W1.3 adds versioned explicit Skill/Tool indexes, deterministic task selection,
15/15 active limits, selected Skill summaries, and Broker-enforced active Tools.
W1.4 composes the versioned Task Plan, host-observation Verifier, bounded
Recoverer/re-plan revisions, trace evidence, and lifecycle event stream into the
single shared Service/Engine path. Legacy simulated/direct execution paths were
removed. W1.5 adds the workspace-visible `agent-folder/bin` and `output` trays,
task-scoped checkpoints, versioned artifact manifests, SHA-256 references,
Provider profile references, and bin-only explicit cleanup. W1.6 provides JSON
schema v1, bounded argv/stdin input,
stdout/stderr separation, stable exit codes, signal cancellation, hashed
idempotency keys, and deterministic non-interactive mutation denial. W1.7 adds
the complete deterministic task fixture suite, migrated-workspace compatibility,
headless TUI/CLI smoke, corrected product documentation, and release/rollback
notes. Work 1 is complete on the host release path; do not enable side-effecting
Automation or SubAgent execution as an implied next step.

Post-W1.7 debug hardening makes runtime discovery read-only, protects and backs
up Provider environment configuration, removes blocking endpoint probes from
the Setup TUI, rejects nil Provider contracts, stabilizes the Windows symlink
fixture, and binds checksum-verified upgrades to the canonical GitHub remote.

Work 1 finalization adds repeatable built-binary folder simulations on Windows
and Linux/WSL, Linux symlink-escape coverage, Linux/Android ARM64 build gates,
and structured Tool failure observations before Agent recovery. GitHub CI runs
the repository tests, vet, and build on both Windows and Linux. This is the
single-Agent release baseline; Work 2 remains gated.

The owner is pursuing S25+ through a separate path. Cross-build is not physical
acceptance: keep mobile TUI, Thai input/rendering, filesystem, cancellation,
Provider networking, and battery/thermal results as external evidence.

Automation with side effects remains gated until non-interactive approval,
stable machine-readable output, exit codes, and artifact manifests are defined
and tested.
