# Acceptance Test Matrix

| # | Property (directive §6) | Status | Test today / planned |
|---|---|---|---|
| 1 | Core/Runtime do not import optional Block implementations | Implemented | `internal/architecture/deps_test.go` `TestBlockDependencyRules` |
| 2 | Removing a Tool Block does not crash the Runtime | Implemented | `app/composition_test.go` `TestRemovingAnOptionalPackKeepsTheRest`; acceptance "environment manifest composes the Agent" |
| 3 | Removing MCP leaves other Tools usable | Implemented | `TestMCPBlockFailuresAreReportedNotFatal` (dead, hung, wrong tool, id clash); acceptance: dead MCP server skipped, task completes |
| 4 | Removing SubAgent keeps single-Agent operation | Implemented | The pack is optional and in no preset; `TestSubAgentPackIsOptionalAndReported`; every v1.11 test runs without it |
| 15 | A SubAgent cannot exceed its parent | Implemented | `internal/subagent/subagent_test.go`: granted Tools only, no nested delegation, write needs approval, shared budget, no parent context |
| 5 | Missing dependency detected before execution | Implemented | `TestCompositionRules` (unknown pack, runtime version, Provider capability); acceptance: required Block missing → exit 5, no task folder created |
| 6 | Unqualified Provider never enters the Router | Implemented | acceptance "unqualified Provider fails before task creation"; `app/provider_environment_test.go` |
| 7 | A Block cannot raise its own authority | Implemented | `ToolSpec.Validate`; Broker assigns risk; `TestCompositionRules` refuses a Tool above `max_risk` and a pack whose `provides` hides a Tool |
| 8 | A removed Block leaves no task or resource behind | Partial | MCP servers run one process per request and are killed when it ends (`mcp/client.go`, `TestClientCancellationStopsServer`); no long-lived Block resources exist yet |
| 9 | Capability report matches reality | Implemented | `TestReportMatchesWhatTheBrokerHolds`, `TestReportSeparatesDeclaredLoadedAndActive`; acceptance: `boi env --json` states |
| 10 | Benchmark does not regress | Implemented | `scripts/acceptance/benchmark.sh` with `BASE_REF`; gate: success ≥ previous, `RUNS=3` |
| 11 | Native, envelope and fallback share one execution path | Implemented | `native_test.go` |
| 12 | Invalid arguments never reach authorization | Implemented | `TestInvalidNativeArgumentsAreRejectedBeforeAuthorization` |
| 13 | Shortened context is recoverable exactly | Implemented | `context_store_test.go` |
| 14 | An MCP server never sees Provider keys | Implemented | `TestMCPBlockPlugsInServerTools` (fails if `PSC_*` reaches the server) |
