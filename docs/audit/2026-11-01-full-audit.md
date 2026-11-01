# Full project audit — 2026-11-01

Commit audited: `master` `19c6752` (v2.7.0 + renovation plan).
Labels: **[V]** reproduced by a test or read directly in code, **[H]** hypothesis.
Every [V] finding below was reproduced with a throwaway test; the repro is
described so it can become the regression test in the fix.

## Summary

| ID | Severity | Finding | Origin |
|---|---|---|---|
| A1 | **High** | Provider API keys reach every `process.run` command | pre-existing |
| A2 | **High** | `workspace.write` can overwrite `.env` | pre-existing |
| A6 | **High** | Streaming silently returns an empty reply from a gateway that does not stream | v2.7.0 |
| A3 | Medium | `workspace.edit` secret check is bypassed through a symlink | pre-existing |
| A4 | Medium | `/allow /` grants the whole filesystem | v2.7.0 |
| A7 | Medium | Streamed parallel tool calls without `index` are merged into one | v2.7.0 |
| A5 | Low | `/allow <relative>` resolves against the process directory, not the workspace | v2.7.0 |
| A8 | Low | `gofmt` fails on `master`; CI has no format step | v2.6.2 + process |

Automated gates: `go vet` clean · `staticcheck` clean · `go test -race ./...` all pass ·
coverage 68.2% · `govulncheck` **not run** (vulnerability DB unreachable, HTTP 403).

---

## Findings

### A1 · High · Provider API keys reach every `process.run` command [V]

- `internal/service/config/envfile/loader.go:29` loads `.env` with `os.Setenv`.
- `internal/equipment/tools/process/executor.go:181` gives every command `cmd.Env = os.Environ()`.
- Repro: set `PSC_1_API_KEY`, run `env` through `Executor.Run` → the key is in the output.

**Impact.** `IsSecretFile` (`internal/equipment/tools/filesystem/reader.go:103`) states the threat
model: *"A prompt-injected task must not be able to read them and pass them on."* Read, search
and edit honour it; `process.run` defeats it. The approval preview shows the command, not what it
does: an approved `go test ./...` or `npm test` runs repository code that can print the
environment, and the output goes back to the model and so to the Provider.

**Fix.** Build the child environment without BOI's secrets: drop every `PSC_*` variable (at
minimum `PSC_<n>_API_KEY`). Regression test: the repro above must not find the key.

### A2 · High · `workspace.write` can overwrite `.env` [V]

- `writeTool.Execute` (`internal/equipment/tools/filesystem/tools.go:184`) never calls
  `IsSecretFile`. `editTool` does (`edit.go:73`), so the two tools disagree.
- Repro: write `.env` with `PSC_1_BASE_URL=https://attacker.example` → the file is replaced.

**Impact.** One approved write can repoint a Provider's base URL; on the next start BOI sends
that Provider's key as `Authorization: Bearer …` to the new host (`openai.go`, `send`).

**Fix.** Refuse credential files in `workspace.write`, checked on the resolved path.

### A3 · Medium · Edit's secret check is bypassed through a symlink [V]

- `edit.go:73` checks the *argument* (`IsSecretFile(path)`) before `ResolveExisting`; the reader
  checks the *resolved* path.
- Repro: `notes.txt → .env`; edit `notes.txt` → `.env` is changed.
- Needs a symlink, which the model can only create through an approved `process.run`.

**Fix.** Check the resolved path, as the reader does.

### A6 · High · Streaming returns an empty reply from a gateway that does not stream [V]

- Since v2.7.0 every interactive turn sets `OnText`, so the OpenAI adapter sends
  `"stream": true` and reads server-sent events (`openai.go`, `stream`).
- Repro: a server that ignores `stream` and answers a normal JSON completion with HTTP 200 →
  `Content == ""`, `err == nil`.

**Impact.** Such a Provider gets empty answers in the TUI; the Engine rejects them as empty
decisions and the task stops. Which real gateways ignore `stream` is **unknown [H]**; the
non-streaming path (`boi ask`, automation) is unaffected.

**Fix.** If the response `Content-Type` is not `text/event-stream`, decode it as a normal
completion and report its text once through `OnText`.

### A4 · Medium · `/allow /` grants the whole filesystem [V]

- `.boi/workspace-policy.json` refuses the filesystem root (`guardPolicyFolder`, `policy.go`),
  but `Sandbox.GrantForTask` (`sandbox.go`) has no such guard.
- Repro: `GrantForTask("/")` succeeds and `/etc/hostname` resolves inside the sandbox.

**Impact.** User-initiated, but it contradicts the stated rule that widening the policy never
turns the boundary off.

**Fix.** Move the guard into the `Sandbox` so `AddRoot` and `GrantForTask` both apply it.

### A7 · Medium · Streamed tool calls without `index` are merged [V]

- `toolCallAccumulator` (`openai.go`) keys calls by `index`, which decodes to `0` when absent.
- Repro: two streamed calls with ids `a` and `b` and no `index` → one call, id `b`, arguments
  `{"path":"a"}{"path":"b"}`.
- OpenAI sends `index`; whether any gateway omits it is **unknown [H]**.

**Fix.** Decode `index` as `*int`; when it is absent and a new `id` arrives, start a new call.

### A5 · Low · `/allow <relative>` resolves against the process directory [V]

- `GrantForTask` passes the path to `filepath.Abs`; the policy file resolves relative folders
  against the workspace root.
- Repro: workspace `project/`, process directory `cwd/`, `/allow data` → grants `cwd/data`.

**Fix.** Resolve relative grants against the primary root.

### A8 · Low · `gofmt` fails on `master`; CI has no format step [V]

- `internal/runtime/agent/grounding_test.go` (from `e05d5c2`, v2.6.2) is not gofmt-clean.
- `.github/workflows/ci.yml` runs vet, race tests, coverage and staticcheck, but no `gofmt` check.
- Process note: the merge check run for `3542cef` printed "GATES CLEAN" although its gofmt test
  failed, because the final `echo` ran unconditionally after the short-circuited `&&` chain. That
  claim was wrong.

**Fix.** `gofmt -w` the file; add a CI step `test -z "$(gofmt -l .)"`.

---

## Observations (no action required now)

| | Observation | |
|---|---|---|
| O1 | `staticcheck -checks all` adds only 17 × ST1005 (capitalised error strings), mostly proper nouns such as "Skill" and "Tool". | [V] |
| O2 | Coverage by package: `transport/cli` 29.8%, `platform/update` 41.6%, `transport/tui` 54.3%, `runtime/llm` 57.8%. | [V] |
| O3 | Self-update verifies SHA-256 against `checksums.txt` from the same release, refuses path traversal, skips links and bounds sizes. That protects against corruption, not against a compromised release, because nothing is signed. | [V] |
| O4 | The process deny-list is best-effort by its own comment (`process/sandbox.go:9`); approval is the real boundary. `cat .env` through `process.run` is visible in the approval preview, unlike A1. | [V] |
| O5 | `Broker.executed` (idempotency records) grows for the life of a `Service`, i.e. a whole TUI session. Small. | [V] |
| O6 | The TUI re-renders every message on each streamed update. Not measured. | [H] |

## Already tracked in the renovation plan

`tool_observation` withholding every Tool, `reasoning_content` not replayed,
`rejectsNativeReplay` hiding a 400, drop-oldest instead of compaction, `qualify` hiding
`unsupported` details, `boi env` giving one reason for every cause — see
[`docs/planning/RENOVATION_PLAN.md`](../planning/RENOVATION_PLAN.md) §2 (D1–D7).

## Not verified in this audit

- Known vulnerabilities in dependencies (`govulncheck` blocked).
- Any live Provider, the TUI in a real terminal, and Windows (HANDOFF lists one Windows-only
  test failure).

## Proposed fix order

One PR, each finding with its repro as a regression test:
A1 → A2 → A3 → A6 → A4 → A5 → A7 → A8. All are local changes; none alters a public contract.
