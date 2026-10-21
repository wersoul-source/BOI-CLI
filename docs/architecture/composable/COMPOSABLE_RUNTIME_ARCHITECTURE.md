# Composable Runtime Architecture

Status: **Proposed**. Nothing here changes code until approved.

## The one decision

Composition happens **at start**, from a file. An *environment manifest*
names the Blocks an Agent uses; `app` reads it, checks it, and registers what
it names. Hot swap during a task is out of scope (directive rule 6).

```text
environment manifest ──► app (composition root)
                          │ resolve + validate (fail before any task)
                          ▼
     Core ── qualification, Persona, capability profile      (unchanged)
     Runtime ── loop, Broker, approval, verification          (unchanged)
          ▲ ports: port.Tool, llm.Provider, MemoryHook, SkillDoc, (SubAgent: new)
          │
     Blocks ── Tool packs · Skill packs · MCP packs · SubAgent packs
```

Core and Runtime do not learn about manifests. They keep receiving Tools,
Skills and Providers through the ports they already have. Only `app` grows.

## What already fits

| Concern | Existing mechanism | Reused as |
|---|---|---|
| Choose Tools per workspace | `.boi/registry/tools.json` | the Tool section of a manifest |
| Choose Skills per workspace | `.boi/skills` + `skills.json`, `Requires` | the Skill section |
| External tools | `mcp.Tools(server, names, invoker)` | an MCP Block |
| Degrade on missing capability | `ComposeEnvironment`, `Requires` | required/optional rules |
| Boundaries | `deps_test.go` | unchanged; manifests must not add imports |

## What is new

1. **Block manifest** (data, per Block): id, kind, version, provides,
   requires, optional, risk ceiling. See `BLOCK_CONTRACT_PROPOSAL.md`.
2. **Environment manifest** (data, per Agent): which Blocks, which are
   required. Presets (Coding, Research, Minimal) are just manifests.
3. **Resolver** in `app`: loads manifests, checks versions and requires,
   produces one *composition report* before the first task.
4. **SubAgent port** in Runtime: a SubAgent is a Tool-like capability whose
   effect is a delegated task; it gets no authority the parent lacks.

## Invariants

- A Block never grants itself authority: risk and approval come from its
  spec and the host policy, never from the manifest alone. A manifest can
  only *lower* a Block's ceiling.
- A missing **required** Block stops start-up with a reason. A missing
  **optional** Block is reported and skipped.
- Removing any optional Block leaves a working single Agent.
- Qualification gates Providers exactly as today.
