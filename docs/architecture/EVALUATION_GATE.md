# Agent Runtime Evaluation Gate

Phase 2 completes only when the host-owned safety boundary passes the following
automated checks:

- workspace traversal and symlink escape are rejected;
- model-provided risk, approval, timeout, target, and preview fields are rejected
  or replaced by host policy;
- approval is bound to the exact tool-call fingerprint and expires;
- write, process, and MCP actions cannot execute without explicit approval;
- process and MCP execution honor cancellation, timeout, and output bounds;
- provider retry and failover occur only for classified eligible failures;
- steps, tool calls, recoveries, tokens, and wall-clock time remain bounded;
- tool output is returned to the model as untrusted observation data;
- idempotency keys prevent duplicate side effects within one broker lifetime.

`SubagentsEnabled` remains false after this gate. Enabling delegation requires a
separate owner decision, task isolation design, delegated budgets, and acceptance
tests; passing the single-agent gate does not authorize that change.

The workspace sandbox is a filesystem path boundary. Approved processes still
run with the operating-system rights of BOI CLI and are not containerized.
