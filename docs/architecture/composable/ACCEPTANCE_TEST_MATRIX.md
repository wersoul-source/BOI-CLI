# Acceptance Test Matrix

| # | Property (directive §6) | Status | Test today / planned |
|---|---|---|---|
| 1 | Core/Runtime do not import optional Block implementations | Implemented | `internal/architecture/deps_test.go` `TestBlockDependencyRules` |
| 2 | Removing a Tool Block does not crash the Runtime | Implemented | `app/composition_test.go` `TestRemovingAnOptionalPackKeepsTheRest`; acceptance "environment manifest composes the Agent" |
| 3 | Removing MCP leaves other Tools usable | Implemented | `TestMCPBlockFailuresAreReportedNotFatal` (dead, hung, wrong tool, id clash); acceptance: dead MCP server skipped, task completes |
| 4 | Removing SubAgent keeps single-Agent operation | Implemented | The pack is optional and in no preset; `TestSubAgentPackIsOptionalAndReported`; every v1.11 test runs without it |
| 15 | A SubAgent cannot exceed its parent | Implemented | `internal/subagent/subagent_test.go`: granted Tools only, no nested delegation, write needs approval, shared budget, no parent context |
| 5 | Missing dependency detected before execution | Implemented | `TestCompositionRules` (unknown pack, runtime version, Provider capability); acceptance: required Block missing → exit 5, no task folder created |
| 6 | A Provider without a measured profile, or whose completion fails, never enters the Router (ADR-006); a weak but completing Provider is calibrated, not excluded | Implemented for the first part; calibration is Planned (Phase 2) | acceptance "unqualified Provider fails before task creation"; `app/provider_environment_test.go` |
| 7 | A Block cannot raise its own authority | Implemented | `ToolSpec.Validate`; Broker assigns risk; `TestCompositionRules` refuses a Tool above `max_risk` and a pack whose `provides` hides a Tool |
| 8 | A removed Block leaves no task or resource behind | Partial | MCP servers run one process per request and are killed when it ends (`mcp/client.go`, `TestClientCancellationStopsServer`); no long-lived Block resources exist yet |
| 9 | Capability report matches reality | Implemented | `TestReportMatchesWhatTheBrokerHolds`, `TestReportSeparatesDeclaredLoadedAndActive`; acceptance: `boi env --json` states |
| 10 | Benchmark does not regress | Implemented | `scripts/acceptance/benchmark.sh` with `BASE_REF`; gate: success ≥ previous, `RUNS=3` |
| 11 | Native, envelope and fallback share one execution path | Implemented | `native_test.go` |
| 12 | Invalid arguments never reach authorization | Implemented | `TestInvalidNativeArgumentsAreRejectedBeforeAuthorization` |
| 13 | Shortened context is recoverable exactly | Implemented | `context_store_test.go` |
| 14 | An MCP server never sees Provider keys | Implemented | `TestMCPBlockPlugsInServerTools` (fails if `PSC_*` reaches the server) |
| 16 | Every Provider that completes gets a Recipe; none loses Tools only for being weak | Planned (Architecture IV Phase 2) | `ComposeRecipe` table tests; acceptance with the simulated model zoo |
| 17 | Host security invariants: no secret reaches a Tool process or the model context; permissions are always enforced by the host | Partial | `TestCommandsDoNotSeeProviderKeys`, `TestWriteRefusesCredentialFiles`, `TestCredentialFileBehindALinkIsRefused` (v2.7.1) |
| 18 | Tool Replay: a side-effecting Tool never runs again on recovery without approval or a proven idempotency guarantee | Planned (Phase 3) | — |
| 19 | Recipe Rollback: a Recipe version that makes results worse can be rolled back | Planned (Phase 5) | — |
| 20 | Provider Drift: a Provider that changes behaviour gets a new Recipe without damaging a task in progress | Planned (Phase 5) | — |
| 21 | Extension Isolation: an installed Extension cannot read secrets or widen the workspace scope by another channel | Planned (Phase 4) | — |
| 22 | Crash Recovery: a task that dies midway resumes | Planned (Phase 3) | — |
