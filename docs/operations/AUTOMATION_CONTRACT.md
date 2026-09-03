# BOI CLI Automation Contract v1

## Scope

Work 1 automation uses `boi ask --json`. It supports read-only Agent work and
deterministically denies every Tool action that requires approval. There is no
non-interactive mutation-approval flag in v1.

## Input

```text
boi ask <query...> [--json] [--idempotency-key <key>]
<UTF-8 stdin> | boi ask --json [--idempotency-key <key>]
```

- argv is authoritative when at least one query argument exists; stdin is not
  read in that case.
- With no query arguments, stdin must be piped, valid UTF-8, non-empty after
  trimming, and at most 1 MiB.
- The command never prompts for a missing query.
- `--idempotency-key` requires `--json`, accepts 1-128 characters from
  `A-Z a-z 0-9 . _ : -`, and is emitted/persisted only as SHA-256.

## Streams

- In `--json` mode stdout contains exactly one JSON object followed by a
  newline.
- Machine failures are represented inside that JSON object. The main process
  does not repeat an already-reported failure on stderr.
- `--verbose` diagnostics use stderr and never contaminate stdout.
- Human mode writes only the Agent response to stdout; verbose runtime details
  use stderr.

## Result schema

`schema_version` is `1`. Additive fields may be introduced without changing
the version; removing or changing the meaning/type of an existing field
requires a new schema version.

```json
{
  "schema_version": 1,
  "status": "completed",
  "task_id": "task-...",
  "response": "...",
  "stop_reason": "completed",
  "provider": "...",
  "model": "...",
  "usage": {
    "steps": 1,
    "input_tokens": 1,
    "output_tokens": 1,
    "provider_calls": 1,
    "tool_calls": 0,
    "duration_ms": 1
  },
  "manifest": "agent-folder/output/<task-id>/manifest.json",
  "artifacts": [],
  "idempotency_key_hash": "sha256:..."
}
```

Failures set `status` and include:

```json
"error": { "class": "denied", "message": "..." }
```

The Agent Folder manifest remains the durable source of truth. JSON output
references that manifest and does not create a second artifact catalog.

## Exit codes

| Code | Class | Meaning |
|---:|---|---|
| 0 | `completed` | Verified task completion |
| 1 | `internal` | Runtime, Tool, budget, state, or unexpected internal failure |
| 2 | `invalid_input` | Invalid argv, stdin, flag, or idempotency key |
| 3 | `denied` | Approval required/rejected or safety policy blocked execution |
| 4 | `cancelled` | Cancellation or deadline/timeout |
| 5 | `unavailable` | Qualified Provider or required runtime dependency unavailable |
| 6 | `verification_failed` | Host verification did not prove the result |

## Authorization and idempotency

- Read Tools may run automatically when active and Provider-qualified.
- Change, execute, external, and critical Tools cannot receive non-interactive
  approval in v1; they stop without execution.
- The host hashes the automation key and namespaces mutation Tool idempotency
  keys. The current v1 contract does not claim a cross-process response replay
  cache. Because side-effecting automation is disabled, repeated automation
  requests cannot duplicate side effects.
- A future mutation contract must add persistent replay state and explicit
  scoped authorization before enabling side effects.
