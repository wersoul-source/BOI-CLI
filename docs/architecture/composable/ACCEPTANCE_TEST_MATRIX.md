# Acceptance Test Matrix

| # | Property (directive §6) | Status | Test today / planned |
|---|---|---|---|
| 1 | Core/Runtime do not import optional Block implementations | Implemented | `internal/architecture/deps_test.go` `TestBlockDependencyRules` |
| 2 | Removing a Tool Block does not crash the Runtime | Implemented | `app/composition_test.go` `TestRemovingAnOptionalPackKeepsTheRest`; acceptance "environment manifest composes the Agent" |
| 3 | Removing MCP leaves other Tools usable | Proposed | Step 2: environment with and without the MCP Block, same simulation |
| 4 | Removing SubAgent keeps single-Agent operation | Implemented (trivially) | SubAgent is disabled today; step 5 adds a with/without test |
| 5 | Missing dependency detected before execution | Implemented | `TestCompositionRules` (unknown pack, runtime version, Provider capability); acceptance: required Block missing → exit 5, no task folder created |
| 6 | Unqualified Provider never enters the Router | Implemented | acceptance "unqualified Provider fails before task creation"; `app/provider_environment_test.go` |
| 7 | A Block cannot raise its own authority | Implemented | `ToolSpec.Validate`; Broker assigns risk; `TestCompositionRules` refuses a Tool above `max_risk` and a pack whose `provides` hides a Tool |
| 8 | A removed Block leaves no task or resource behind | Proposed | Step 2: MCP process stopped on removal and on exit; checked by process table in test |
| 9 | Capability report matches reality | Proposed | Step 3: report vs `Broker.CapabilityNames()` per environment |
| 10 | Benchmark does not regress | Implemented | `scripts/acceptance/benchmark.sh` with `BASE_REF`; gate: success ≥ previous, `RUNS=3` |
| 11 | Native, envelope and fallback share one execution path | Implemented | `native_test.go` |
| 12 | Invalid arguments never reach authorization | Implemented | `TestInvalidNativeArgumentsAreRejectedBeforeAuthorization` |
| 13 | Shortened context is recoverable exactly | Implemented | `context_store_test.go` |
