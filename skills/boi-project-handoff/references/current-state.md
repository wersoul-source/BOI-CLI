# Current State

## Confirmed implementation

- The owner-approved six Blocks now have explicit package manifests under `internal/block`.
- Conformance tests require exactly six unique valid manifests and prevent concrete Blocks from importing one another directly.
- Repository was restructured around `internal/app`, transports, Agent, Provider, Tool, Memory, Persona, Skill, Workspace, Config, and Platform boundaries.
- TUI and `boi ask` use `internal/agent.Service`.
- The bounded Engine implements Observe, Decide, Authorize, Act, Verify, Recover, typed stops, step/tool/token/time budgets, and cancellation.
- Provider Router classifies failures and performs bounded retry/failover.
- Capability Broker owns Tool risk, approval, timeout, preview, and execution policy.
- Workspace read is auto-eligible; write, process, and registered MCP actions require exact approval.
- TUI has an interactive approval panel. Non-interactive CLI rejects calls requiring approval.
- Workspace Sandbox rejects traversal and symlink escape. It is not OS isolation.
- Adversarial tests cover untrusted Tool observations, approval enforcement, idempotency, and disabled Subagents.

## Implemented but not connected to the main Agent path

- Skill loading and prompt helpers exist, but Agent Service does not select or inject Skills.
- Planner and Reviewer types exist, but the main Service does not compose them.
- MCP client and Broker registration exist, but no complete discovery/Library workflow is connected.
- Legacy `agent.Loop` has simulated no-provider behavior; the active Service returns `ErrNoProvider` instead.

## Concept changes not implemented

- Replace six selectable Personas with one Core Persona, `boi`.
- Add first-run Agent naming for TUI/GUI and persist Agent instance identity.
- Add Provider capability tests and an environment qualification profile.
- Add Local Skill/Tool indexes and registries with an active maximum of 15 each.
- Add offline Foundation bundle plus BOI MCP/Library routing.
- Move existing behavior behind the new six-Block boundaries and composition contracts.
- Add Agent Folder output/bin tray and artifact manifest.
- Add gated SubAgent Market workflow.

## Known documentation drift

- README still advertises six Personas and simulated mode without API keys.
- Archived and lifecycle documents may describe earlier architecture as complete.
- `Verify` currently defaults to basic Tool status validation unless a Verifier is injected.
- Engine supports Recoverer, but Agent Service does not currently inject one.

## Verification at handoff preparation

- `go test ./...`: passed.
- `go vet ./...`: passed.
- Targeted Agent approval, recovery, untrusted-observation, Provider retry/failover, and disabled-Subagent tests: passed.
- `GOOS=android GOARCH=arm64 go build ./cmd/boi`: passed on Windows.
- Physical S25+ runtime: not yet verified.

## Verification after Phase H1 skeleton

- `go test ./internal/block/...`: passed.
- `go test ./...`: passed.
- `go vet ./...`: passed.
- Runtime behavior migration: intentionally not started.
