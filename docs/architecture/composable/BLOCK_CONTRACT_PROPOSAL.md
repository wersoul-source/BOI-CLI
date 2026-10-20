# Block Contract Proposal

Status: **Proposed**. Requires an ADR before code (directive rule 9).

## Block manifest (`block.json`, one per Block)

```json
{
  "schema_version": 1,
  "id": "coding-tools",
  "kind": "tool-pack",
  "version": "1.0.0",
  "provides": ["workspace.read", "workspace.write", "process.run"],
  "requires": {"runtime": ">=1.8", "capabilities": ["tool_calling"]},
  "max_risk": "execute",
  "source": {"type": "builtin"}
}
```

| Field | Meaning | Checked by |
|---|---|---|
| `kind` | `tool-pack`, `skill-pack`, `mcp-pack`, `subagent-pack` | resolver |
| `provides` | capability names the Block registers | resolver; must match what is actually registered |
| `requires.runtime` | Runtime version range | resolver |
| `requires.capabilities` | Provider capabilities from qualification | resolver, against the profile |
| `max_risk` | highest risk any provided Tool may have | resolver; a Tool above it is refused |
| `source` | `builtin`, `mcp` (command + args), `files` (Skill dir) | `app` |

A manifest cannot raise a Tool's risk or lower its approval. `port.ToolSpec`
stays the single source of truth for each Tool (ADR-001).

## Environment manifest (`.boi/environment.json`)

```json
{
  "schema_version": 1,
  "name": "coding",
  "blocks": [
    {"id": "workspace-tools", "required": true},
    {"id": "coding-tools", "required": false},
    {"id": "github-mcp", "required": false}
  ]
}
```

No file means today's behavior: the built-in Tools and the existing indexes.

## SubAgent port (Runtime)

```go
type SubAgent interface {
    Spec() port.ToolSpec // name, schema, risk = at least the parent's highest delegated risk
    Delegate(ctx context.Context, task string, limits EngineLimits) (*AgentResult, error)
}
```

Rules: the child runs under the parent's authorizer and a budget carved out
of the parent's; it cannot register Blocks; its result is verified like any
Tool observation. Organization, factory and watchdog features are out of
scope until this port passes its security gates.

## Compatibility

v1.7 workspaces keep working: absent manifests resolve to the built-in Tool
set, the current indexes and no MCP or SubAgent Blocks.
