# BOI Agent Runtime Contract

This document defines the host-side contract for the TUI-first Agent runtime.
Model output is untrusted intent. It cannot execute a tool or authorize its own
request.

## Lifecycle

```text
observe -> decide -> authorize -> act -> verify
                \                 |
                 -> recover <------
```

Every active phase may stop with a typed reason. Step, time, token, provider,
and tool-call budgets are enforced by the runtime loop rather than by the
transport.

## Tool call

A Tool Call is a proposal containing an ID, tool name, purpose, arguments,
target, expected result, user-visible preview, risk, approval class, timeout,
and optional idempotency key.

The approval request stores a SHA-256 fingerprint of the complete Tool Call.
Changing its arguments, target, preview, risk, or timeout invalidates the
approval.

Risk and approval rules:

| Risk | Minimum behavior |
|---|---|
| read | May be automatically allowed by workspace policy |
| change | Explicit confirmation |
| execute | Explicit confirmation |
| external | Explicit confirmation and data-egress preview |
| critical | Critical confirmation or denial |

## Tool result

A Tool Result is a host observation linked to one Tool Call. It records typed
status, bounded output, changed paths, evidence, error classification, and
timing. An Agent may claim a side effect only when the Tool Result confirms it.

## Approval

Approval states are requested, approved, rejected, expired, and cancelled.
Approval is exact, single-action, and time-bounded. The current TUI panel:

- shows purpose, tool, target, risk, and preview;
- accepts explicit `A`/ `Y` approval;
- accepts `R`/ `N` rejection;
- uses Esc or Ctrl+C to cancel the task;
- deliberately does not accept Enter as approval;
- expires automatically through the TUI tick lifecycle.

Write, process, and registered MCP execution pass through the Capability
Broker. The approval transport returns a decision for the exact request and
never rebuilds a call from display text.

## Stop reasons

The runtime uses typed stop reasons including completed, needs approval,
rejected, cancelled, timeout, max steps, budget exhausted, tool failure,
provider failure, verification failure, safety blocked, and invalid state.

## Current boundary

The bounded Kernel loop, host-owned Capability Broker, model proposal parser,
and TUI approval state machine are connected. Read-only workspace capabilities
may run automatically. Write, process, and registered MCP calls require an
exact expiring approval. The filesystem boundary does not provide
operating-system or container isolation for an approved process.
