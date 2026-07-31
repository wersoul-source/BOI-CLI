# BOI CLI -- Architecture & Command Map

> Version: v0.1.0
> Updated: 31 July 2026
> By: Kampun (Kampun) -- BOI Family

---

## Table of Contents

1. [Full Command Tree](#full-command-tree)
2. [Internal Package Architecture](#internal-package-architecture)
3. [Package Dependency Graph](#package-dependency-graph)
4. [Package Descriptions](#package-descriptions)
5. [Layer Architecture](#layer-architecture)
6. [Data Flow Between Packages](#data-flow-between-packages)

---

## Full Command Tree

```
boi
|
+-- (no args) ---------------------------------------> TUI Mode
|    |-- Splash screen (boiLogoDOS + workspace info)
|    |-- Bubbletea full-screen chat interface
|    |-- Tab: cycle persona, Ctrl+N: newline, Ctrl+Q: quit
|
+-- ask [query] -------------------------------------> AI Agent (ReAct loop)
|    |-- --persona, -p  string  "kamkaew"    Persona (boi|kamkaew|kampun|dang|don|kine)
|    |-- --steps, -s    int     "15"         Max agent steps per query
|    |-- --verbose, -v  bool    "false"      Show steps, tokens, time
|    |
|    +-- Example: boi ask "debug login" -p dang -v
|    +-- Without .env: simulated response mode (no real AI)
|    +-- Exit codes: 0=ok, 1=error, 4=provider error
|
+-- run [command] -----------------------------------> Shell Command Execution
|    |-- --verbose, -v  bool    "false"      Verbose output
|    |-- --dir, -d      string  (workspace)  Working directory override
|    |
|    +-- Sandbox safety: basic allow/deny list
|    +-- Inherits shell environment
|
+-- init --------------------------------------------> Initialize Workspace
|    |-- --force, -f    bool    "false"      Overwrite existing config
|    |
|    +-- Creates: .boi/config.yaml, .boi/personas/, .boi/skills/, .boi/memory/
|    +-- Creates: .boi/.gitignore
|    +-- Error if .boi/ exists (use --force to overwrite)
|
+-- config ------------------------------------------> View Configuration
|    |-- --all, -a      bool    "false"      Show full YAML with masked API keys
|    |
|    +-- Summary mode: Provider, Model, Log Level, Workspace, API Key count
|    +-- Full mode: YAML dump of entire config
|    +-- Loads from: .boi/config.yaml
|
+-- persona -----------------------------------------> Persona Management
|    |-- list                                      List all personas
|    |   |-- Shows: NAME, MODEL, TEMP, DESCRIPTION
|    |   |-- Active persona marked with *
|    |   |-- Loads from: .boi/personas/*.yaml
|    |
|    |-- switch <name>                             Switch active persona
|    |   |-- Validates name against loaded personas
|    |   |-- Updates: .boi/config.yaml persona field
|    |   |-- Shows: Name, Description, Model, Temperature
|    |   |-- Example: boi persona switch dang
|    |
|    +-- init                                      Copy default persona YAMLs
|        |-- Copies from embedded defaults to .boi/personas/
|        |-- Skips existing files (no overwrite)
|
+-- skill -------------------------------------------> Skill Management
|    |-- list                                      List loaded skills
|    |   |-- Shows skills from .boi/skills/*.skill.md
|    |   |-- Built-in: git, web
|    |
|    |-- init                                      Initialize default skills
|    |   |-- Creates: git.skill.md, web.skill.md
|    |
|    +-- show <name>                               Show skill definition
|        |-- Prints full markdown content
|        |-- Example: boi skill show git
|
+-- memory ------------------------------------------> Phantom DB Management
|    |-- search <query>                            Full-text search
|    |   |-- Keyword match on key (+3.0) and content (+1.0)
|    |   |-- Sorted by score descending, limit 10
|    |   |-- Shows: type, key, score, content preview
|    |
|    |-- stats                                     Memory statistics
|    |   |-- Total entries, facts, patterns, solutions
|    |   |-- File count, storage path
|    |
|    |-- save <type> <key> <content>               Manual save
|    |   |-- Args: memory type (fact/pattern/solution)
|    |   |--       memory key (slug for lookup)
|    |   |--       memory content (free text)
|    |   |-- Example: boi memory save fact "port" "Uses port 3055"
|    |
|    |-- repomap                                   Repository map
|    |   |-- Scans project structure (skips .boi, .git, node_modules, etc.)
|    |   |-- Shows: file count by extension, total size, largest files
|    |
|    +-- init                                      Initialize memory.md
|        |-- Creates .boi/memory.md with template sections
|        |-- Sections: Architecture, Conventions, Known Issues, Key Decisions
|
+-- weight -------------------------------------------> Weight Engine
|    +-- explain <id>                              Weight breakdown
|        |-- Shows per-dimension explanation
|        |-- Dimensions: truth, confidence, importance, recency, usage
|        |-- Example: boi weight explain mem_1785403325978479400
|
+-- help --------------------------------------------> Auto-generated by Cobra
|
+-- --version ---------------------------------------> Prints "boi version 0.1.0"

EXIT CODES:
  0  Success
  1  General error
  2  Invalid arguments / flag
  3  Config not found (run 'boi init' first)
  4  Provider error (check .env PSC_* variables)

ENVIRONMENT VARIABLES:
  BOI_HOME              Override .boi/ directory location
  BOI_LOG_LEVEL         Override log level (debug/info/warn/error)
  PSC_1_NAME            Provider 1 name (openai|anthropic|deepseek|ollama)
  PSC_1_API_KEY         Provider 1 API key
  PSC_1_BASE_URL        Provider 1 base URL (optional, for proxies)
  PSC_1_MODEL           Provider 1 model (gpt-4.1-mini|claude-sonnet-5|etc)
  PSC_2_* ... PSC_4_*   Fallback providers 2 through 4
```

---

## Internal Package Architecture

```
cmd/boi/
|-- main.go                  Entry point, TUI dispatch
|-- tui.go                   Bubbletea program setup

internal/
|-- cli/                     Cobra command definitions (L1)
|   |-- root.go              boi base command, subcommand registration
|   |-- agent.go             boi ask -- persona binding + agent loop setup
|   |-- run.go               boi run -- shell command wrapper
|   |-- init.go              boi init -- workspace initialization
|   |-- config.go            boi config -- config display
|   |-- persona.go           boi persona list/switch/init
|   |-- skill.go             boi skill list/init/show
|   |-- memory.go            boi memory search/stats/save/repomap/init
|   |-- weight.go            boi weight explain

|-- config/                  Configuration system (L1)
|   |-- config.go            Load/save .boi/config.yaml
|   |-- defaults.go          Default values + fallback logic

|-- workspace/               Workspace management (L1)
|   |-- workspace.go         DetectRoot(), GetBoiDir()
|   |-- scanner.go           Project structure scanning

|-- logger/                  Structured logging (L1)
|   |-- logger.go            slog-based logger setup
|   |-- levels.go            DEBUG/INFO/WARN/ERROR constants

|-- command/                 Shell command execution (L1)
|   |-- executor.go          Shell command runner
|   |-- sandbox.go           Allow/deny list for safety

|-- agent/                   Agent core (L3)
|   |-- loop.go              ReAct loop controller
|   |-- types.go             AgentState, AgentStep, AgentResult
|   |-- planner.go           Task decomposition
|   |-- executor.go          Tool execution engine
|   |-- reviewer.go          Quality review + self-correction
|   |-- subagent.go          Sub-agent delegation

|-- persona/                 Persona system (L4)
|   |-- types.go             Persona struct
|   |-- registry.go          Registry (Load/Get/List)
|   |-- loader.go            YAML file loader
|   |-- defaults.go          Embedded default YAMLs (go:embed)

|-- skill/                   Skill system (L5)
|   |-- types.go             Skill struct
|   |-- registry.go          Skill registry (Load/Get/List)
|   |-- loader.go            SKILL.md file loader
|   |-- runtime.go           Skill execution runtime

|-- mcp/                     MCP protocol (L5 extension)
|   |-- client.go            MCP client integration

|-- llm/                     Provider Supply Chain (L6)
|   |-- provider.go          Provider interface + response types
|   |-- router.go            Linear fallback router
|   |-- streaming.go         SSE streaming helper
|   |-- factory/
|   |   |-- factory.go       LoadProvidersFromEnv()
|   |-- providers/
|       |-- openai.go        OpenAI / OpenAI-compatible provider
|       |-- anthropic.go     Anthropic Claude provider

|-- memory/                  Phantom DB (L7)
|   |-- store.go             File-based JSON memory store
|   |-- hook.go              MemoryHook (BeforeTurn/AfterTurn)
|   |-- prefetch.go          Prefetch/InjectMemory/ReWeight/SetScore
|   |-- context.go           ContextManager (token budget tracking)
|   |-- compaction.go        Token budget compaction
|   |-- extractor.go         Fact extraction from conversations
|   |-- entity.go            weight.Entity adapter for MemoryEntry
|   |-- repomap.go           Codebase structure scanner
|   |-- claudemd.go          CLAUDE.md hierarchy loader
|   |-- nudge.go             Periodic maintenance triggers

|-- weight/                  Weight Engine (L7 companion)
|   |-- engine.go            Engine (Compute/ComputeAndExplain/ApplyDecay)
|   |-- types.go             Entity interface + Weights struct
|   |-- policy.go            WeightPolicy + DefaultPolicy()
|   |-- explain.go           Explain() + WeightExplanation

|-- tui/                     Terminal UI (L2)
    |-- chat.go              Chat viewport + message rendering
    |-- splash.go            Splash screen + workspace info
    |-- status.go            Top bar status (persona, level, idle/busy)
    |-- input.go             Multi-line input with key bindings
    |-- help.go              Keyboard shortcut overlay
    |-- styles.go            Lipgloss style definitions
    |-- eightart.go          ASCII art banner rendering
```

---

## Package Dependency Graph

```
                        +-----------+
                        |  cmd/boi  |  Entry point
                        +-----+-----+
                              |
                              v
          +-------------------+-------------------+
          |                                       |
          v                                       v
   +-----------+                          +-------------+
   |  cli/     |  (Cobra commands)        |  tui/       |  (Bubbletea)
   +-----+-----+                          +------+------+
         |                                       |
         v                                       v
   +-----------+                          +-----------+
   | workspace |  DetectRoot, GetBoiDir   |  splash   |
   +-----------+                          +-----------+
         |
         v
+--------+--------+
|                  |
v                  v
+--------+    +---------+
| config |    | persona |  YAML loaders
+--------+    +---------+
|                  |
v                  v
+--------+    +---------+
| skill  |    | memory  |  <-- Runtime data
+--------+    +----+----+
|                  |
v                  v
+--------+    +---------+
|  mcp   |    | weight  |  <-- Scoring engine
+--------+    +---------+
                  |
                  v
            +-----------+
            |   agent   |  ReAct loop
            +-----+-----+
                  |
                  v
            +-----------+
            |    llm    |  PSC Router + Providers
            +-----------+

DEPENDENCY DIRECTION:
  cmd/boi -> cli, tui
  cli     -> workspace, config, persona, skill, memory, weight, agent, llm
  tui     -> workspace, persona, skill, memory, agent
  agent   -> persona, llm, memory
  memory  -> weight (entity adapter)
  llm     -> (standalone, no internal deps)
  persona -> (standalone, YAML only)
  skill   -> mcp (extension)
  weight  -> (standalone, pure math)
  config  -> (standalone, YAML I/O)
```

---

## Package Descriptions

### `cmd/boi/` -- Entry Point

| File | Purpose |
|------|---------|
| `main.go` | Detects if no args (launches TUI) or has args (dispatches to Cobra CLI). |
| `tui.go` | Initializes Bubbletea program: loads workspace, personas, memory, skills, then starts splash -> chat flow. |

**Responsibility:** Top-level routing only. No business logic.

---

### `internal/cli/` -- Command Layer (L1)

All Cobra command definitions. Each file registers commands on `rootCmd`. This is the interface between user and system.

| File | Commands Registered | Dependencies |
|------|---------------------|--------------|
| `root.go` | `boi` (base) | none |
| `agent.go` | `boi ask` | workspace, persona, llm/factory, agent, memory |
| `run.go` | `boi run` | workspace, command |
| `init.go` | `boi init` | config, logger |
| `config.go` | `boi config` | workspace, config |
| `persona.go` | `boi persona {list,switch,init}` | workspace, config, persona |
| `skill.go` | `boi skill {list,init,show}` | skill |
| `memory.go` | `boi memory {search,stats,save,repomap,init}` | workspace, memory |
| `weight.go` | `boi weight explain` | workspace, memory, weight |

---

### `internal/config/` -- Configuration (L1)

| File | Purpose |
|------|---------|
| `config.go` | Struct definition (`Config`), YAML load/save via `gopkg.in/yaml.v3`, Viper integration path |
| `defaults.go` | `Default()` returns a Config with sensible defaults (provider: "", model: "", log_level: "info") |

**Config structure:**
```yaml
provider: ""
model: ""
log_level: "info"
persona: "kamkaew"
api_keys: {}
```

---

### `internal/workspace/` -- Workspace Detection (L1)

| File | Purpose |
|------|---------|
| `workspace.go` | `DetectRoot()` -- walks up directory tree looking for `.boi/` or `.git/`. `GetBoiDir()` returns `.boi/` path. |
| `scanner.go` | Project structure scanning utilities. |

**Algorithm:** Linear walk up from CWD, checking each parent for marker directories. Falls back to CWD if no markers found.

---

### `internal/logger/` -- Structured Logging (L1)

| File | Purpose |
|------|---------|
| `logger.go` | `New()` creates a `slog.Logger` with INFO level default. Supports level override via `BOI_LOG_LEVEL`. |
| `levels.go` | Log level constants (DEBUG, INFO, WARN, ERROR). |

---

### `internal/command/` -- Shell Execution (L1)

| File | Purpose |
|------|---------|
| `executor.go` | Wraps `os/exec` for shell command execution with timeout and output capture. |
| `sandbox.go` | Basic allow/deny list for dangerous commands. Checks against a configurable list. |

---

### `internal/agent/` -- Agent Core (L3)

The central execution engine. Orchestrates the ReAct loop:

| File | Purpose |
|------|---------|
| `types.go` | `AgentState` (current state), `AgentStep` (one iteration), `AgentResult` (final output) |
| `loop.go` | `Loop.Run()` -- main controller. Manages state, calls LLM, checks completion, triggers memory hooks. |
| `planner.go` | Task decomposition -- breaks complex tasks into sub-steps. |
| `executor.go` | Tool execution engine -- maps tool names to functions. |
| `reviewer.go` | Quality review -- evaluates response before returning. |
| `subagent.go` | Sub-agent delegation -- spawns child agents for subtasks. |

**Loop contract:**
```
1. Init AgentState with task + persona
2. BeforeTurn: inject memory context
3. For step 1..MaxSteps:
   a. Build messages: system_prompt + user_query
   b. callLLM(messages) -> response
   c. If final answer: AfterTurn, return AgentResult
   d. If tool request: continue loop
4. Exceeded steps -> error
```

---

### `internal/persona/` -- Persona System (L4)

| File | Purpose |
|------|---------|
| `types.go` | `Persona` struct: Name, Model, Temperature, SystemPrompt, Description |
| `registry.go` | In-memory registry: `Load()`, `Get(name)`, `List()`, `Register()` |
| `loader.go` | YAML loader: reads `.boi/personas/*.yaml` files. Falls back to embed. |
| `defaults.go` | `DefaultPersona()` and embedded YAML files via `//go:embed defaults/` |

**Persona contract:**
```yaml
name: dang
model: gpt-4.1-mini
temperature: 0.2
system_prompt: "You are Dang, an elite debug specialist..."
description: "Debug & Code Specialist"
```

---

### `internal/skill/` -- Skill System (L5)

| File | Purpose |
|------|---------|
| `types.go` | `Skill` struct: Name, Description, Instructions, Tools |
| `registry.go` | In-memory registry: load from `.boi/skills/*.skill.md` |
| `loader.go` | Markdown parser: extracts frontmatter + instructions from SKILL.md files |
| `runtime.go` | Skill execution engine: loads skill context and provides tool implementations |

**Skill format (.skill.md):**
```markdown
---
name: git
description: Git operations assistant
tools:
  - git_status
  - git_diff
  - git_commit
---
# Git Skill
You are a git operations specialist...
```

---

### `internal/mcp/` -- MCP Protocol Client (L5 extension)

| File | Purpose |
|------|---------|
| `client.go` | MCP client implementation using `mark3labs/mcp-go`. Handles JSON-RPC transport. |

---

### `internal/llm/` -- Provider Supply Chain (L6)

| File | Purpose |
|------|---------|
| `provider.go` | `Provider` interface: `Name()`, `Complete()`, `SupportModel()`. `CompletionRequest`/`CompletionResponse` structs. |
| `router.go` | `NewRouter()` -> linear fallback chain. Retries HTTP errors, fails hard on auth/config errors. |
| `streaming.go` | SSE streaming handler for real-time response display. |
| `factory/factory.go` | `LoadProvidersFromEnv()` -- reads PSC_1..PSC_4 env vars, creates provider instances. |
| `providers/openai.go` | OpenAI (and OpenAI-compatible) provider. Maps to `/v1/chat/completions`. |
| `providers/anthropic.go` | Anthropic Claude provider. Maps to `/v1/messages`. |

**Provider interface:**
```go
type Provider interface {
    Name() string
    Complete(ctx context.Context, req CompletionRequest) (*CompletionResponse, error)
    SupportModel(model string) bool
}
```

---

### `internal/memory/` -- Phantom DB (L7)

| File | Purpose |
|------|---------|
| `store.go` | File-based JSON store. `Open()`, `Save()`, `Query()`, `SearchMemory()`, `Stats()`, `Close()`, `CleanExpired()`. Thread-safe via `sync.RWMutex`. |
| `hook.go` | `MemoryHook` -- lifecycle manager. `BeforeTurn()` injects context, `AfterTurn()` extracts + saves. Nudges every 10 turns. |
| `prefetch.go` | `Prefetch()` -- keyword search + XML format. `InjectMemory()` -- convenience wrapper. `ReWeight()` -- conflict resolution. `SetScore()` -- manual score update. |
| `context.go` | `ContextManager` -- tracks conversation messages + token budget. `AddMessage()`, `IsOverBudget()`, `Summarize()`. |
| `compaction.go` | Token budget compaction -- summarizes older messages to stay under limit. |
| `extractor.go` | `Extractor` interface + `SimpleExtractor`. Extracts structured facts from agent conversations. |
| `entity.go` | `MemoryEntry` adapter implementing `weight.Entity` interface. Maps memory fields to weight dimensions. |
| `repomap.go` | `ScanRepo()` -- walks project tree, skips known directories, returns file count + sizes. |
| `claudemd.go` | Loads CLAUDE.md files from project hierarchy for context injection. |
| `nudge.go` | Periodic maintenance triggers (compaction, cleanup). |

---

### `internal/weight/` -- Weight Engine (L7 companion)

| File | Purpose |
|------|---------|
| `engine.go` | `Compute()` -- weighted average of 5 dimensions. `ComputeAndExplain()` -- returns full breakdown. `ApplyDecay()` -- time-based score reduction. |
| `types.go` | `Entity` interface (5 dimensions). `Weights` struct with `.Sum()` method. |
| `policy.go` | `WeightPolicy` struct. `DefaultPolicy()` with hardcoded weights (Truth=0.35, Confidence=0.25, Usage=0.20, Importance=0.10, Recency=0.10). |
| `explain.go` | `Explain()` generates `WeightExplanation` with per-dimension breakdown, reasons, and contributions. |

---

### `internal/tui/` -- Terminal UI (L2)

Bubbletea-based full-screen terminal interface.

| File | Purpose |
|------|---------|
| `chat.go` | `ChatModel` -- viewport with message history. `AddMessage()` renders user/agent/system/error messages with lipgloss styles. |
| `splash.go` | `SplashModel` -- displays ASCII art logo + workspace info + "Press Enter to start". |
| `status.go` | Top bar: persona name, level tier, idle/busy status, time. |
| `input.go` | Multi-line text input at bottom with key bindings. |
| `help.go` | Overlay with keyboard shortcuts (`/help` command). |
| `styles.go` | Lipgloss style definitions for all UI elements (colors, borders, padding). |
| `eightart.go` | ASCII art rendering utilities for the banner. |

---

## Layer Architecture

```
+=============================================================================+
|  L1: CLI & INTERFACE     Cobra CLI + Flags + Help                          |
|                          internal/cli/*, internal/config/*,                 |
|                          internal/workspace/*, internal/logger/*           |
+=============================================================================+
|  L2: TUI                 Bubbletea full-screen interface                    |
|                          internal/tui/*                                     |
+=============================================================================+
|  L3: AGENT CORE          ReAct Loop + Planner + Reviewer + SubAgent        |
|                          internal/agent/*                                   |
+=============================================================================+
|  L4: PERSONA SYSTEM      6 Personas: boi, kamkaew, kampun, dang, don, kine |
|                          internal/persona/*                                 |
+=============================================================================+
|  L5: SKILL SYSTEM        Skill Runtime + Loader + Registry + MCP           |
|                          internal/skill/*, internal/mcp/*                  |
+=============================================================================+
|  L6: PSC (Providers)     OpenAI + Anthropic + DeepSeek + Ollama            |
|                          internal/llm/*                                     |
+=============================================================================+
|  L7: MEMORY & WEIGHT     Phantom DB + Context + Weight Engine               |
|                          internal/memory/*, internal/weight/*               |
+=============================================================================+
|  L8: EVOLUTION           GEPA Trace + Pattern Detection + Scoring           |
|                          (future: internal/evolution/*)                     |
+=============================================================================+
```

**Dependency rule:** Higher layers depend on lower layers. Lower layers NEVER import higher layers.

---

## Data Flow Between Packages

```
User Input
    |
    v
cli/agent.go (Cobra handler)
    |
    +-> workspace.DetectRoot()              -> root path
    +-> persona.LoadDir() + registry.Get()  -> active Persona
    +-> llm/factory.LoadProvidersFromEnv()  -> []llm.Provider
    +-> memory.Open() + NewMemoryHook()     -> memory hook
    |
    v
agent.NewLoop(persona, providers, hook)
    |
    v
loop.Run(ctx, query)
    |-> hook.BeforeTurn(query)
    |   |-> store.InjectMemory()            -> memory context string
    |   |       |-> store.SearchMemory()    -> keyword match
    |   |       |-> weight.Engine.Compute() -> score ranking
    |   |
    |-> loop.buildSystemPrompt()            -> persona.SystemPrompt
    |-> loop.callLLM(messages)
    |   |-> llm.NewRouter(providers).Complete()
    |       |-> provider.Complete()         -> LLM API call
    |
    |-> hook.AfterTurn(query, response)
        |-> extractor.Extract()             -> []ExtractedFact
        |-> store.Save(entry)               -> .boi/memory/*.json
        |-> hook.Nudge() (every 10 turns)
```

---

## Convention Map

| Convention | Value |
|------------|-------|
| Module path | `github.com/boi-family/boi-cli` |
| Go version | 1.24+ |
| CLI library | `github.com/spf13/cobra` |
| TUI library | `github.com/charmbracelet/bubbletea` |
| Styling | `github.com/charmbracelet/lipgloss` |
| Config format | YAML (`gopkg.in/yaml.v3`) |
| Logging | `log/slog` (stdlib) |
| MCP | `github.com/mark3labs/mcp-go` |
| SQLite | `modernc.org/sqlite` (zero-dep) |
| Memory storage | JSON files on disk (no DB server) |
| Workspace marker | `.boi/` directory |
| Config location | `.boi/config.yaml` |
| Persona location | `.boi/personas/*.yaml` |
| Skill location | `.boi/skills/*.skill.md` |
| Memory location | `.boi/memory/*.json` |
| Binary size target | < 10 MB |
| Build command | `go build -o bin/boi ./cmd/boi` |
| Cross-compile | `GOOS=<os> GOARCH=<arch> go build` |

---

*End of CLI_ARCHITECTURE.md*
