# BOI Agent Kernel Loop

Phase 2 Task 6 introduces one bounded Agent kernel in `internal/agent.Engine`.
Both the interactive Agent Service and the compatibility CLI Loop execute
through this kernel.

## State flow

```text
observe -> decide -> verify -> stopped
             |
             v
         authorize -> act -> verify -> decide
             |         |       |
             +------ recover --+
```

Every transition is checked by `CanTransition`. Invalid transitions stop with
`invalid_state`.

## Runtime ports

The kernel depends on narrow interfaces rather than concrete transports or
tools:

| Port | Responsibility |
|---|---|
| `Decider` | Return a final response or a validated Tool Call |
| `Authorizer` | Allow, reject, or return a pending approval |
| `Actor` | Execute one already-authorized Tool Call |
| `Verifier` | Check a final response or Tool Result |
| `Recoverer` | Decide whether a bounded retry is justified |

The default authorization policy permits only read-risk Tool Calls explicitly
marked `auto`. `Service` supplies an interactive Authorizer for TUI use and a
rejecting Authorizer for non-interactive CLI use.

## Bounds

The Engine enforces:

- maximum decision steps;
- maximum Tool Calls;
- maximum recoveries;
- token budget;
- total task timeout;
- individual Tool Call timeout;
- context cancellation.

`Service` connects the kernel to a host-owned Capability Broker. Workspace
reads may run automatically; write, process, and registered MCP capabilities
require an expiring approval bound to the exact Tool Call fingerprint.
