# BOI CLI Handoff

This is the entry point for Kampun or any Agent continuing BOI CLI.

## Repository

- Remote: `https://github.com/wersoul-source/BOI-CLI.git`
- Primary branch: `master`
- Published handoff baseline: `f5c0934`
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
removed. Continue with W1.5 Agent Folder.

The owner is pursuing S25+ through a separate path. Keep physical mobile results
as external acceptance evidence and do not block the main six-Block sequence on
that work.

Automation with side effects remains gated until non-interactive approval,
stable machine-readable output, exit codes, and artifact manifests are defined
and tested.
