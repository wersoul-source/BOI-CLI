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
- Agent Service composes a stable versioned Task Plan, deterministic host Verifier, bounded Recoverer, and trace evidence into Engine.
- Recovery creates a Plan revision before re-deciding and remains bounded by recovery, step, Tool, token, and time limits.
- Runtime emits task, phase, Tool, approval, Tool Result, verification, and stop events; TUI consumes this stream.
- Legacy simulated Loop, direct shell/file Executor, and placeholder Reviewer were removed; Broker is the only Tool execution path.
- `agent-folder/bin` and `agent-folder/output` are the single workspace-level task trays composed into both TUI and CLI.
- Every Agent run receives a collision-safe Task ID, credential-free lifecycle checkpoint, and final versioned manifest.
- Completed manifests catalog deliverables with size and SHA-256 evidence under output; failed/cancelled manifests remain under bin.
- Cleanup is dry-run by default, explicit when applied, and structurally restricted to bin; output has no automatic deletion path.
- `boi ask --json` emits one schema-v1 result object with stable Task/manifest/artifact references and stable exit-code classes.
- Automation accepts argv or bounded piped UTF-8 stdin, never prompts for missing input, and keeps verbose diagnostics on stderr.
- Automation idempotency keys are host-hashed and namespaced; raw keys are not persisted.
- Work 1 Automation is read-only: approval-required mutation is deterministically denied without waiting or execution, and no cross-process replay-cache claim is made.
- Work 1 task fixtures cover explanation, repository inspection, report creation, approved write, rejection, cancellation, Provider failure, and bounded recovery.
- Migrated workspaces receive only missing registry indexes; existing config and legacy Persona files are preserved and loose Skills remain unexposed.
- README, command reference, and release/rollback notes describe the one-Persona, qualified-Provider product contract.

## Implemented but not connected to the main Agent path

- MCP client and Broker registration exist, but no complete discovery/Library workflow is connected.

## Concept changes not implemented

- Add Service health and MCP routing without an offline-operation guarantee.
- Add gated SubAgent Market workflow.

## Documentation boundary

- README and the CLI command reference match Work 1.
- Archived and lifecycle documents may still describe historical architecture and must not override the live executable or Work 1 plan.

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

## Verification after Work 1 Phase W1.4

- Plan validation/determinism, recovery revision, read-back verification, and safe recovery classification tests: passed.
- First-class Skill Call tests prove full instructions load only from the active task set and do not grant Tool authority.
- Shared Service approval/write loop proves Plan -> Authorize -> Act -> Verify -> Complete and lifecycle event emission.
- Full repository tests, vet, native build, and Android ARM64 cross-build: passed.
- Go race detector was unavailable in the current Windows toolchain because CGO is disabled; Engine events clone mutable Plan/Result/Evidence data before delivery.

## Verification after Work 1 Phase W1.5

- Primary-tray, task-ID collision, checkpoint, completed/failed routing, artifact hash, secret-exclusion, and bin-only cleanup tests: passed.
- Agent Service composition test proves task paths reach the Provider as workspace-relative data and lifecycle events/final results reach the recorder.
- Full repository tests, vet, native build, and Android ARM64 cross-build: passed.

## Verification after Work 1 Phase W1.6

- argv/stdin, UTF-8, input limit, idempotency-key validation, JSON schema, single-object output, exit mapping, and reported-error tests: passed.
- Non-interactive mutation test proves `needs_approval`, hashed key propagation, no wait, and no filesystem side effect.
- Built-binary smoke proves empty piped stdin emits one `invalid_input` JSON object, no stderr, and process exit code 2.
- Full repository tests, vet, native build, and Android ARM64 cross-build: passed.

## Verification after Work 1 Phase W1.7

- Deterministic Work 1 end-to-end and migrated-workspace compatibility fixtures: passed.
- Headless TUI state/render and built CLI contract smokes: passed.
- Full repository tests, vet, native build, and cross-build targets: passed.
- Physical S25+ runtime and live billable Provider qualification: not verified by the host path.
