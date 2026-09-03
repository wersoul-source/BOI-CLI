# Work 1 Release and Rollback Notes

## Release candidate scope

The commit containing this file is the Work 1 host-verified release candidate.
It delivers one Core Persona, a user-named Agent, qualified Provider routing,
bounded Skill/Tool registries, one controlled Runtime, one Agent Folder, TUI
approval, and read-only Automation.

SubAgent execution, full MCP Library discovery, side-effecting Automation, and
offline-first operation are not part of this release candidate.

## Upgrade and migration

1. Preserve `.boi` and `agent-folder` before replacing a binary.
2. Build or install the candidate and run `boi doctor`.
3. Run `boi provider qualify <name>` for every Provider intended for routing.
4. Run `boi registry init` and inspect `boi registry list`.
5. Start `boi` and confirm the stored Agent name, Provider, and TUI state.

Runtime initialization creates only missing Skill and Tool indexes. Existing
configuration, legacy Persona files, and existing indexes are not overwritten.
Legacy Persona identities remain stored but runtime Persona is fixed to `boi`.
Loose Skill or Tool files remain inactive until explicitly registered.

## Acceptance evidence

| Gate | Result |
|---|---|
| Unit, adversarial, and Work 1 task fixtures | Passed on Windows host |
| Migrated-workspace compatibility fixture | Passed on Windows host |
| CLI parser/JSON smoke | Passed on built Windows binary |
| TUI state/render smoke | Passed headlessly; not a physical-terminal test |
| Native build and vet | Passed on Windows host |
| Android ARM64 cross-build | Passed; build evidence only |
| Live third-party Provider qualification | Not run automatically; may be billable |
| Physical S25+ Termux/TUI/runtime | Not verified in this release path |

## Rollback

1. Stop the running BOI process.
2. Copy `.boi` and `agent-folder` to a safe backup location. Never delete
   `agent-folder/output` as part of rollback.
3. Restore the previous binary or check out the pre-W1.7 baseline `f3db72b`.
4. Run `boi doctor` and inspect `.boi/config.yaml` before starting tasks.

W1.7 migration is additive: it creates missing registry indexes and does not
rewrite legacy Persona/config files. Therefore rollback does not require a
destructive schema downgrade. New output manifests can remain as historical
artifacts even when an older binary does not consume them.

## Release blockers outside host acceptance

- Physical S25+ validation must observe `boi version`, `boi doctor`, one TUI
  query, one `boi ask` query, Thai rendering/input, cancellation, storage paths,
  Provider networking, and practical battery/thermal behavior.
- Side-effecting Automation requires a persistent replay and scoped machine
  authorization contract.
- SubAgent requires isolated budgets and delegated-authority evaluation.
