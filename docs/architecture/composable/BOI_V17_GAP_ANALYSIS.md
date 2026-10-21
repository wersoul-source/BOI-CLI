# BOI v1.7 Gap Analysis

Baseline: branch `agent-suite-v1.7` (commit `9f571d7`, on top of master `497b229`).
Status keys: **Implemented** (code + test), **Partial** (some code, not on the
main path or not general), **Proposed** (no code), **Blocked** (intentionally off).

## 1. Thin Core

| Requirement | Status | Evidence |
|---|---|---|
| Core holds identity, Persona, qualification, capability profile | Implemented | `internal/core/identity.go`, `core/persona`, `core/qualification.go` |
| No agent-type logic in Core | Implemented | Core imports only `runtime/llm` and the standard library; `internal/architecture/deps_test.go` |
| Qualification checks real behavior before Router entry | Implemented | `Qualifier.Run`, 8 probes, retry once; `app.QualifiedProviders`; acceptance "unqualified Provider fails before task creation" |
| Reuse of unexpired qualification | Partial | Profiles are cached per user (`~/.config/boi/provider-profiles`) and invalidated by probe-suite version or target change, but have **no age limit** |

## 2. Stable Runtime

| Requirement | Status | Evidence |
|---|---|---|
| Loop, Broker, authorization, verification, recovery, lifecycle | Implemented | `runtime/agent/engine.go`, `broker.go`, `authorization.go`, `runtime_verifier.go`, `runtime_recoverer.go`, `lifecycle.go` |
| Runtime not tied to a Block type | Implemented | Broker knows Tools only via `port.Tool`; deps test forbids spine → plug imports |
| Native + envelope + fallback to one execution path | Implemented | `service.go` (`nativeDecision`, `ParseDecision`, `rejectsNativeReplay`), `native_test.go` |
| Argument validation before authorization | Implemented | `port.ToolSpec.ValidateArguments` in `Broker.Prepare`; `schema_test.go` |
| Context store and exact read-back | Implemented | `context_store.go`, `context_store_test.go` |

## 3. Replaceable Blocks

| Block kind | Status | Evidence / gap |
|---|---|---|
| Tool (compile time) | Implemented | `port.Tool`; `app.BuiltinTools`; adding one needs a rebuild |
| Tool set chosen per workspace without code | Implemented | `.boi/registry/tools.json` (`enabled`, priority, tags), 15 active max; `capability.Select` |
| Skill (file based, run time) | Implemented | `.boi/skills` + `skills.json`; `Requires` rejects a Skill whose Tools are missing (`capability/registry.go:172`) |
| MCP | Partial | Client and adapter exist (`equipment/tools/mcp`), tested; **not wired**: no config, no discovery in `app.BuildAgent` |
| Plugin | Proposed | No code |
| SubAgent | Blocked | `subagent.Enabled = false`; `Delegate` returns `ErrSubagentsDisabled`; no port |
| Block manifest | Partial | `block.Manifest` records owner, purpose, boundary, status. It has no version, dependencies, capabilities or activation rules and nothing reads it at run time |
| Packs (Tool/Skill/MCP bundles) | Proposed | No grouping above a single index entry |

## 4. Graceful degradation

| Requirement | Status | Evidence |
|---|---|---|
| Failed Tool-calling probe → completion-only Agent | Implemented | `ComposeEnvironment`, `Broker.SetToolCallingAllowed` |
| Native rejected → envelope | Implemented | `rejectsNativeReplay` |
| Skill without its Tools is rejected, not loaded broken | Implemented | `Requires` check |
| Missing optional Block reported, Runtime continues | Partial | Only for Skills and Provider capabilities; no notion of optional Tool/MCP/SubAgent Blocks |
| Capability report | Partial | `boi doctor`, `boi provider list`, `boi registry list` exist separately; no single report of what is active and why |

## 5. Dependency boundaries

| Item | Status | Evidence |
|---|---|---|
| Spine never imports a plug Block | Implemented | deps test |
| Plug Block never imports another plug Block | Implemented | deps test |
| Transport reaches Blocks only through `app` | Partial | 13 allowlisted edges remain (`deps_test.go:36-49`) |

## 6. Measured behavior (Cloudflare Workers AI, llama-3.3-70b, 3 runs × 6 tasks)

| Version | Success | Notes |
|---|---|---|
| v1.5 | 12–14 / 18 | Model reads files with `process.run`, stops on approval |
| v1.6 | 18 / 18 (twice), 17 / 18 | `deep_read` 21k–35k tokens |
| v1.7 (batched reads, 4 KB preview) | 16 / 18 | `multi_read` 2 calls vs 4; `deep_read` 1/3 → preview change reverted |

Open risk: `deep_read` is the least stable task in every version. The model
never uses `context.expand` `match`; it pages through the file.

## 7. Summary

The fixed spine, the Tool contract, qualification and degradation of Provider
capabilities are real and tested. Composition exists **inside** one Block kind
(Tool index, Skill files) but not **across** kinds: there is no machine-read
Block manifest, MCP is not wired, Plugin and SubAgent have no port, and there
is no environment manifest that says which Blocks a given Agent uses.
