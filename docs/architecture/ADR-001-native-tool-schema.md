# ADR-001: Adaptive Native Tool Schema

Status: accepted for v1.6 · Scope: `block/port`, `runtime/agent`, `runtime/llm`,
`service/provider/adapters`, `core` qualification. No new Block.

## Context

v1.5 sent Tools as text (`<boi-action>` envelopes). Models trained for native
function calling were forced into a custom format, and the host had no
machine-readable argument contract. v1.6 adds native tool calling. Doing so
raises three questions: where the schema lives, how it reaches different
Providers, and who checks what a model sends back.

## Decision

1. **One canonical contract.** `port.ToolSpec` is the only source of truth:
   name, description, argument schema (`Parameters`, or derived from
   `Usage`), risk, approval and timeout. Every Provider format is generated
   from it; nothing is copied.
2. **One supported schema subset.** `ToolSpec.ProviderSchema()` reduces the
   schema to `type, description, properties, required, items, enum,
   additionalProperties(bool)`. Other keywords (`$ref`, `oneOf`, `format`, …)
   are dropped before a definition leaves the host, so a Provider never
   receives a keyword it may reject, and the host never advertises a rule it
   does not enforce.
3. **The Runtime validates, not the Provider.** `ToolSpec.ValidateArguments`
   runs in `Broker.Prepare` for native calls and envelopes alike, before
   authorization. Order: parse → schema validation → normalization to
   `ToolCall` → authorization → execution → verification. A Provider's strict
   mode is never treated as validation or as permission.
4. **Capability decides the format, measured not assumed.** Native mode needs
   the adapter (`llm.ToolCaller`), every Router candidate, and a passing
   `native_tool_schema` probe. An endpoint that accepts tool definitions but
   rejects the replayed tool messages (HTTP 400/422) drops the task back to
   envelopes. Model names and parameter counts play no part.
5. **Strict mode is off.** No adapter sends `strict: true`. It is enabled per
   Provider only after a probe proves the endpoint accepts it.

## Deferred, with the trigger that reopens each

| Item | Why not now | Reopen when |
|---|---|---|
| Compact / Standard / Extended profiles | One adapter and four simple Tools; the current output already is the compact form | A second adapter family or a Tool with nested arguments ships |
| `input_examples`, strict mode | No Provider in use is verified to support them | A probe confirms support on a real endpoint |
| Contract version field | `ProbeSuiteVersion` already invalidates stale profiles; the Go module versions the contract | A Tool is loaded from outside the binary with its own schema |
| Batch size above 5 | Benchmarks so far use at most 3 parallel reads | Benchmark shows tasks hitting the limit |

## Consequences

- An argument the schema does not declare is now rejected (`additionalProperties:
  false` on schemas derived from `Usage`). The model gets the reason and
  retries; nothing is authorized or executed. MCP Tools keep an open object
  schema until their own schemas are imported.
- Profiles from v1.5 do not load (probe suite v2); every Provider must be
  qualified again.
- Adding a Provider family means one adapter that maps `llm.ToolDefinition`
  and `llm.ToolCallRequest`; the Runtime does not change.

## Evidence

Cloudflare Workers AI, `@cf/meta/llama-3.3-70b-instruct-fp8-fast`, same six
read-only tasks (`scripts/acceptance/benchmark.sh`):

| Version | Success | Time |
|---|---|---|
| v1.5 (envelope only) | 4/6 | 142.1 s |
| v1.6 (native + batch + context store) | 5/6 | 59.7 s |

Single run; repeat with `RUNS=3` before treating it as a result.
