# BOI CLI Handoff

This is the entry point for Kampun or any Agent continuing BOI CLI.

## Repository

- Remote: `https://github.com/wersoul-source/BOI-CLI.git`
- Primary branch: `master`
- Baseline before this handoff: `1efafde`
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

## Immediate next gate

Validate the repository on the physical S25+ using the mobile runbook. Cross-build has passed on Windows for `android/arm64`, but physical execution, TUI rendering, storage permissions, cancellation, and network behavior remain unverified.

Automation with side effects is not ready until non-interactive approval, stable machine-readable output, exit codes, and artifact manifests are defined and tested.
