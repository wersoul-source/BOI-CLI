# Composition Roadmap

Each step is one PR, keeps v1.7 behavior when no manifest exists, and ships
with tests, a benchmark run and a rollback (revert the PR; no data migration).

## Step order and why

The directive's Phase 1 lists eight components. Built at once, they would be
a framework with one tenant. The order below adds one real Block kind per
step, so every new piece of the contract is exercised by a real Block.

| Step | Version | Change | Done when | Status |
|---|---|---|---|---|
| 0 | v1.7 | Gap analysis, this roadmap; single-Agent stability (`workspace.search`) | Approved by the owner; `deep_read` 3/3 | Implemented |
| 1 | v1.8 | ADR-002 Block contract; pack manifests (compiled in) and resolver in `app`; `.boi/environment.json` | Removing the manifest's optional Tool Block leaves the rest working; a missing required Block fails before any task | Implemented (ADR-002) |
| 2 | v1.8 | MCP Block: an environment manifest entry starts an MCP server and registers its Tools | An MCP Block can be added and removed without code; its Tools are approval-gated; a dead server is reported and skipped | Implemented (v1.9) |
| 3 | v1.9 | Composition report (`boi env` / doctor section): active, skipped and why | Report matches what the Broker actually registered in every test environment | Proposed |
| 4 | v1.9 | Presets: Minimal, Coding, Research as environment manifests | Each preset starts and passes its benchmark subset | Proposed |
| 5 | v2.0 | SubAgent port, single SubAgent behind it | Removing the Block returns to single-Agent; child cannot exceed parent authority or budget | Proposed |
| 6 | v2.x | Agent packs, coordination | Requires step 5 security gate | Blocked |
| 7 | v2.x | Organization, factory, watchdogs | Requires a product spec | Blocked |
| 8 | later | External registry, signing, install/remove | Requires steps 1–4 in use | Proposed |

## Not on this roadmap

- Hot swap during a task.
- New logic in Core.
- Mandatory install of any optional Block.

## Before step 1

Single-Agent reliability is still the weakest point: `deep_read` passes 1–3
of 3 depending on the run. Composition multiplies whatever the single Agent
does. Fixing that task's stability is a precondition, not a parallel track.
