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
- Runtime Persona is fixed to the embedded Core Persona `boi` in TUI and `boi ask`.
- First TUI startup creates a versioned, user-named Agent identity at `.boi/agent.yaml`; subsequent starts load it.
- TUI splash and status distinguish Agent name from Provider metadata and the fixed Core Persona.
- Legacy Persona files are preserved; switching to other Personas now returns an explicit compatibility error.
- `boi provider qualify <name>` runs the versioned Provider behavioral probe suite and stores a credential-free profile under `.boi/provider-profiles`.
- Provider candidates without a passing completion profile are excluded from the Agent Router.
- Tool and Skill capability exposure is the fail-closed intersection of every qualified failover candidate.
- `.boi/registry/skills.json` and `tools.json` are the only Local Registry entrypoints; loose files are not exposed.
- Deterministic selection records installed, eligible, selected, active, disabled, and rejected states with reasons.
- Runtime active sets are capped at 15 Skills and 15 Tools; Broker rejects inactive Tools even when registered.
- Agent Service receives selected Skill summaries only; full instructions are restricted to the task's active Skill set.

## Implemented but not connected to the main Agent path

- Skill loading and prompt helpers exist, but Agent Service does not select or inject Skills.
- Planner and Reviewer types exist, but the main Service does not compose them.
- MCP client and Broker registration exist, but no complete discovery/Library workflow is connected.
- Legacy `agent.Loop` has simulated no-provider behavior; the active Service returns `ErrNoProvider` instead.

## Concept changes not implemented

- Add Service health and MCP routing without an offline-operation guarantee.
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

## Verification after Work 1 Phase W1.1

- `go test ./...`: passed.
- Core Identity round-trip, validation, Thai-name, and restart tests: passed.
- TUI Agent name/Core Persona contract tests: passed.
- CLI Persona compatibility smoke: passed; legacy Persona selection is rejected.

## Verification after Work 1 Phase W1.2

- Deterministic Provider qualification, profile round-trip, reproducibility, and fail-closed composition tests: passed.
- Provider qualification CLI registration smoke: passed.
- Live third-party Provider qualification was not run automatically because it performs billable/network API calls; users run it explicitly per configured Provider.

## Verification after Work 1 Phase W1.3

- Registry schema, round-trip, state, deterministic selection, Context budget, and 15/15 limit tests: passed.
- Loose-Skill isolation and active-only full-instruction tests: passed.
- Broker inactive Tool and registered-but-inactive MCP Tool tests: passed.
- CLI `registry init/list/add` commands implemented; parser and repository test suite pass.
