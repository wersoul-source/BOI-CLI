# ADR-005: Completion evidence

Status: accepted for v2.4. Scope: `runtime/agent` (Engine, AgentResult),
`transport` (automation envelope, TUI text). The Broker, Authorizer and the
Tool contract are unchanged.

## Context

A task could end as `completed` while the model's answer said "fixed" and no
file had changed. In the Cloudflare coding benchmark llama-4-scout did this in
10 of 12 runs. Counting Tool calls does not fix it: a model can read a file and
then offer the fix as text, delegate to a helper that does nothing, or make
every edit fail and still claim success.

## Decision

1. **Evidence is recorded by the host.** An *effect* is a Tool call that is not
   a read, is not a delegation, succeeded, and passed verification. An
   `effectLedger` per task records effect calls and the changed paths. It is
   shared with SubAgent children through the task budget, so a child's real
   change reaches the parent and the delegate call itself proves nothing.
2. **Outcome.** Every `AgentResult` carries `Outcome` and `ChangedPaths`:
   `answered` (no Tool ran), `no_effect` (Tools ran, none had an effect),
   `executed` (a command or outside call ran), `changed` (a file changed).
   The automation envelope exposes both. They come from the ledger, never from
   the model's words.
3. **Challenge, once.** In an interactive run where the model can change files
   or run commands, and the task or the answer names an existing workspace
   file, an answer is *unevidenced* when there is no effect and the model made
   no Tool call, made a change attempt that did not take effect, or offered
   code in a fenced block. The answer is sent back once with the host's
   reason; the model acts or answers again. The challenge uses one recovery,
   so there is no loop.
4. **Notice.** If the model answers again without an effect, the run still
   completes and the result carries `Notice`: "Nothing was changed or run in
   this task." The TUI shows it under the answer.
5. **Left alone:** plain conversation (no workspace file named), reading a file
   and answering in prose, non-interactive runs (`boi ask`, `--json`) and
   read-only environments. They cost no extra provider call.

## Consequences

- The host never decides what the user meant; it only refuses to let an answer
  pass as work when nothing was changed. File Q&A that includes a code block
  and no effect costs one extra provider call and carries the notice.
- A task that ends `completed` with `answered`/`no_effect` and a Notice is
  visibly different from one that ends `completed` with `changed`.
- The check works the same for native tool calling and text envelopes: it sits
  in the Engine, above the decision format.

## Limits

- Only files named by their path are recognised. A task without a path that
  claims a change in prose is not challenged; `Outcome` still says
  `answered`.
- A model that makes a change and then describes a different one is not
  detected; verification covers each Tool call, not the story.

## Rollback

Revert the PR. `Outcome`, `ChangedPaths` and `Notice` disappear; nothing else
depends on them.
