# BOI CLI Handoff

This is the entry point for Kampun or any Agent continuing BOI CLI.

## Repository

- Remote: `https://github.com/wersoul-source/BOI-CLI.git`
- Primary branch: `master`
- Published handoff baseline: `67805d1`
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

## Owner-approved direction

BOI Agent Suit has six Blocks: Service, Core, Various Equipment, Runtime, Agent Folder, and SubAgent. It has one Core Persona, `boi`; the user names their Agent instance during first TUI/GUI onboarding. Core evaluates each Provider Model and composes an environment from a bounded active registry of at most 15 Skills and 15 Tools. Foundation capabilities must work offline; additional capabilities may be discovered through BOI MCP and Library routing.

## Current continuation gate

Phase H1 introduces the six package boundaries and conformance tests under
`internal/block`. These packages declare ownership only; existing behavior has
not been moved into them. Continue with Phase H2 Core identity and Provider
qualification contracts before migrating concrete services.

The owner is pursuing S25+ through a separate path. Keep physical mobile results
as external acceptance evidence and do not block the main six-Block sequence on
that work.

Automation with side effects remains gated until non-interactive approval,
stable machine-readable output, exit codes, and artifact manifests are defined
and tested.
