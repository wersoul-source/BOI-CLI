# Acceptance Log

Real-machine evidence for BOI Agent Suite. Each entry records what ran, where,
and what it proved. CI and local fake-provider runs are not entries here.

## 2026-10-20 — First real Provider round trip on Linux

| Field | Value |
|---|---|
| Commit | `79a9bd2` (branch `agent-suite-v1.5`) |
| Machine | Ubuntu 24.04 VM, kernel 6.8.0-147, x86_64, Go 1.24.2 |
| Provider | Cloudflare Workers AI, OpenAI-compatible endpoint |
| Model | `@cf/meta/llama-3.3-70b-instruct-fp8-fast` |
| Command | `bash scripts/acceptance/cloudflare_smoke.sh` |

| Stage | Result |
|---|---|
| Unit, race and architecture tests | passed |
| Built-binary folder simulation | 9/9 scenarios passed |
| Provider qualification | 7/7 probes passed (native tool schema unsupported by design) |
| `boi ask --json` read task | completed: read `note.txt`, answered the marker, 2 steps, 1 tool call, 17.6 s |

Earlier the same day the run failed with `BOI action envelope must be the
entire response`: the model wrapped a correct envelope in prose. Fixed in
`79a9bd2`; the rerun above passed.

Also recorded:

- Home machine, Linux Lite (kernel 7.0), Go 1.26.8: stages 1-3 passed.
- OpenCode Console free models refuse non-OpenCode clients
  (`FreeTierError`); they cannot be used as a BOI Provider.

Not yet proven: approval-gated writes through the TUI, multi-step tasks on a
real model, and daily use.
