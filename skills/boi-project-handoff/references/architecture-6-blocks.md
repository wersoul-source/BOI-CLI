# BOI Agent Suit Six Blocks

## 1. Service

A semi-SDK of optional facilities that Core may compose. It detects the environment, resolves dependencies, exposes offline Foundation capabilities, reports Provider/MCP availability, and discovers possible communication routes. It informs Core; it does not decide for Core.

## 2. Core

Owns the BOI constitution, the single `boi` Persona, Agent identity, Provider Model tests, capability profiling, Tool/Skill/Knowledge calibration, environment composition, and qualification gate.

Provider Model -> Core tests -> Capability Profile -> composed Agent environment.

## 3. Various Equipment

Contains MCP, Tools, Plugins, Skills, Memory, Knowledge, Commands, Planner, Status, and TUI/GUI onboarding equipment. Existence does not grant activation or authority. Core selects equipment; Runtime Registry activates it; Broker controls authority.

## 4. Runtime

Owns lifecycle, state machine, scheduling, Context runtime, Provider routing, Capability Broker, Approval, execution, verification, recovery, budgets, cancellation, events, checkpoints, and shutdown/resume.

## 5. Agent Folder

The single output tray for generated work.

```text
agent-folder/
|-- bin/       temporary, logs, drafts, checkpoints, failed and recovery data
`-- output/    user deliverables, reports, code, data, media and manifests
```

Each completed task should produce one output directory with a summary, artifacts, evidence, and manifest. Evidence is output verification material, not a Library Router responsibility.

## 6. SubAgent

A gated market of diverse SubAgents. The Agent must load the Block-specific Skill before using it.

```text
subagents/
|-- skill/     procedures for discovery, validation, delegation and result checking
|-- index/     navigation to local or remote sources
`-- packages/  installed SubAgent manifests, contracts, tests and Skills
```

Subagents never inherit ambient authority. Each delegation requires an explicit task, Context, Skills, Tools, Workspace scope, budget, stop conditions, output contract, source trust, and verification.

## Active capability rule

- Library capacity is not limited to 15.
- Core selects a runtime working set based on task, Provider capability, environment, network, Context budget, and policy.
- Active Local Registry maximum: 15 Skills and 15 Tools.
- Unregistered files are not exposed to the Agent automatically.
