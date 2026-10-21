# ADR-003: Single SubAgent as a Block

Status: **accepted** by the owner; implemented in v2.0.0. Gates 1–5 pass;
gate 6 (benchmark on the reference Provider) is deferred by the owner.
Scope: a new Runtime port, a `subagent` pack in `app`, engine budget sharing.
Core unchanged.

## Context

`internal/subagent` was a disabled stub. The directive asks for SubAgents
that range from one helper to whole Agent organizations, removable without
touching the Runtime, never able to grant themselves authority. The first
step is one SubAgent, with the authority and budget rules fixed before any
richer form is built on them.

## Decision

### 1. A SubAgent is a Tool the parent calls

The `subagent` pack adds one Tool, `agent.delegate(task, tools)`. The parent
model calls it like any other Tool: the Broker validates the arguments,
authorizes the call and verifies the result. Without the pack, the Tool does
not exist and the Agent is the single Agent of v1.11.

### 2. The child never has more authority than the parent

| Rule | How it is enforced |
|---|---|
| Child Tools ⊆ parent's active Tools | `tools` is checked against the parent Broker's offered set at call time; anything else is refused before the child starts |
| Child cannot delegate | `agent.delegate` is never in the child's set (depth 1) |
| Child cannot add Blocks | the child gets a Service built from already-registered Tools; it has no composition step |
| Same approval path | the child uses the parent's authorizer: in the TUI every write or command still asks the user; in `boi ask` it is denied, exactly as for the parent |
| Delegate's own risk | the highest risk among the requested Tools; approval `confirm` if any requested Tool needs approval, `auto` only when all are automatic reads |
| Same Providers | the child uses the parent's qualified Router; no other model can be reached |

### 3. The child's budget is carved out of the parent's

A shared budget travels in the task context (like the context store). The
child's provider calls, tokens and tool calls are charged to it, so the
parent's limits (64k tokens, 8 Tool calls, 2 minutes) cover the whole task.
The child also has its own smaller ceiling (6 steps, 4 Tool calls, 16k
tokens) and inherits the parent's deadline.

### 4. The child's result is an observation

The child returns its response, stop reason, changed paths and evidence. The
parent treats it as untrusted data. A child that does not complete is a
failed Tool result, handled by the normal recovery rules. Changed paths and
evidence are recorded in the parent's task manifest.

### 5. Isolation

The child sees only the delegated task and its own observations: not the
parent's conversation, memory or Skills. Its context store is its own.

## Out of scope

Agent packs, coordination between SubAgents, organizations, factories and
watchdogs (roadmap steps 6–7) stay blocked until this design has passed its
gates.

## Gates before the pack can be enabled

1. Child cannot call a Tool outside its granted subset (test with a model
   that tries).
2. Child cannot call `agent.delegate`.
3. A child write in `boi ask` is denied; in the TUI it raises the same
   approval prompt as a parent write.
4. Parent plus child never exceed the parent's token, Tool-call or time
   budget.
5. Removing the pack leaves every v1.11 test and the benchmark unchanged.
6. Benchmark task with a delegated sub-task passes 3/3 on the reference
   Provider.

## Consequences

- The engine gains a shared budget, used by the child; a parent without a
  SubAgent behaves exactly as before.
- The SubAgent is a normal Block: `boi env` reports it, presets may include
  it, and `environment.json` can leave it out.

## Rollback

Revert the PR; workspaces that listed the `subagent` pack must remove it.
