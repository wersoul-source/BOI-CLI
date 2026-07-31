# BOI CLI -- System Operational Flow

> Version: v0.1.0
> Updated: 31 July 2026
> By: Kampun (Kampun) -- BOI Family

---

## Overview

This document describes the complete operational flow of BOI CLI -- every step from user input to response, including sub-flows for TUI mode, memory management, and the weight engine.

```
User types: boi ask "debug login bug"
    |
    v
+-------------------------------------------------------------+
|                  BOI CLI SYSTEM FLOW                          |
+-------------------------------------------------------------+
|                                                              |
|  [1] CLI Parser (Cobra)                                      |
|      Parse command + flags + persona                         |
|      |                                                        |
|      v                                                        |
|  [2] Workspace Detection                                     |
|      Find .boi/ -> load config + personas + skills           |
|      |                                                        |
|      v                                                        |
|  [3] Persona Loader                                          |
|      Load active persona -> system prompt + model            |
|      |                                                        |
|      v                                                        |
|  [4] Memory Hook (Phantom DB)                                |
|      Prefetch relevant memories -> inject context            |
|      |                                                        |
|      v                                                        |
|  [5] Weight Engine                                           |
|      Score + rank memories by truth/confidence               |
|      |                                                        |
|      v                                                        |
|  [6] PSC Router                                              |
|      Select provider -> try primary -> fallback              |
|      |                                                        |
|      v                                                        |
|  [7] LLM Call                                                |
|      Send prompt + context + memories -> stream back         |
|      |                                                        |
|      v                                                        |
|  [8] Agent Loop (ReAct)                                      |
|      Thought -> Act -> Observe -> repeat                     |
|      |                                                        |
|      v                                                        |
|  [9] Post-Processing                                         |
|      Extract facts -> save to Phantom DB -> reweight         |
|      |                                                        |
|      v                                                        |
|  [10] Response                                               |
|       Display to user (TUI or CLI)                           |
|                                                              |
+-------------------------------------------------------------+
```

---

## Step-by-Step Detail

### [1] CLI Parser (Cobra)

**What happens:**
The user's command string is parsed by Cobra to determine the command, subcommand, flags, and arguments.

**Modules involved:**
| File | Role |
|------|------|
| `cmd/boi/main.go` | Entry point -- checks `len(os.Args)`. If no args, routes to TUI. Otherwise calls `cli.Execute()` |
| `internal/cli/root.go` | Root Cobra command (`boi`) with version and subcommand registration |
| `internal/cli/agent.go` | `boi ask` -- defines `--persona/-p`, `--steps/-s`, `--verbose/-v` flags |
| `internal/cli/run.go` | `boi run` -- shell command execution |
| `internal/cli/init.go` | `boi init` with `--force/-f` |
| `internal/cli/config.go` | `boi config` with `--all/-a` |
| `internal/cli/persona.go` | `boi persona {list,switch,init}` |
| `internal/cli/memory.go` | `boi memory {search,stats,save,repomap,init}` |
| `internal/cli/weight.go` | `boi weight explain <id>` |
| `internal/cli/skill.go` | `boi skill {list,init,show}` |

**Data flow:**
```
os.Args -> main.go -> (no args) ? runTUI() : cli.Execute()
                                    |
                                    v
                              Cobra dispatch
                              (cmd, flags, args)
```

**Error handling:**
- Invalid command: Cobra default "unknown command" message, exit code 2
- Missing required arg: Cobra returns error, printed to stderr
- Flag parse error: Cobra prints usage, exit code 1

---

### [2] Workspace Detection

**What happens:**
The system locates the project root by walking up the directory tree looking for `.boi/` or `.git/`. Once found, it loads configuration, persona YAML files, and skill definitions.

**Modules involved:**
| File | Role |
|------|------|
| `internal/workspace/workspace.go` | `DetectRoot()` -- walks up from CWD looking for `.boi/` then `.git/`. `GetBoiDir()` returns `.boi/` path |
| `internal/workspace/scanner.go` | Project structure scanning |
| `internal/config/config.go` | Load/save `.boi/config.yaml` |
| `internal/config/defaults.go` | Default configuration values |
| `internal/persona/registry.go` | Persona registry -- Load/Get/List |
| `internal/persona/loader.go` | YAML loader for persona profiles |
| `internal/skill/registry.go` | Skill registry |
| `internal/skill/loader.go` | SKILL.md loader |

**Data flow:**
```
os.Getwd()
    |
    v
DetectRoot() -- walk up tree
    |-- check: .boi/ exists? -> return dir
    |-- check: .git/ exists?  -> return dir
    |-- parent == dir?        -> return cwd (fallback)
    |
    v
GetBoiDir(root) -> ".boi/"
    |-- config.yaml     -> config.LoadFrom()
    |-- personas/*.yaml -> persona.LoadDir()
    |-- skills/*.md     -> skill registry
    |-- memory/*.json   -> Phantom DB
```

**Error handling:**
- No `.boi/` found: falls back to CWD, auto-creates `.boi/` if `boi init` was run
- Config parse error: uses defaults from `config.Default()`
- Missing persona dir: uses embedded default personas via `go:embed`
- Missing skill dir: continues without skills (graceful degradation)

---

### [3] Persona Loader

**What happens:**
The active persona is loaded from `.boi/config.yaml` (or defaults to `kamkaew`). The persona provides the system prompt, model binding, temperature, and behavioral instructions.

**Modules involved:**
| File | Role |
|------|------|
| `internal/persona/registry.go` | Registry -- in-memory map of persona name -> Persona struct |
| `internal/persona/loader.go` | Load YAML files from `.boi/personas/` directory |
| `internal/persona/types.go` | Persona struct with Name, Model, Temperature, SystemPrompt, Description |
| `internal/persona/defaults.go` | Embedded default persona YAMLs via `go:embed defaults/` |
| `.boi/personas/{boi,kamkaew,kampun,dang,don,kine}.yaml` | Per-project persona overrides |
| `.boi/config.yaml` | Stores active persona name under `persona:` key |

**Persona bindings:**
| Persona | Default Model | Temp | Role |
|---------|--------------|------|------|
| boi | claude-sonnet-5 | 0.4 | Architecture & Strategy |
| kamkaew | gpt-4.1-mini | 0.5 | Runtime Orchestrator (default) |
| kampun | claude-sonnet-5 | 0.3 | Root Cause & Pattern Analysis |
| dang | gpt-4.1-mini | 0.2 | Debug & Code Specialist |
| don | gpt-4.1-nano | 0.5 | Documentation Specialist |
| kine | gpt-4o | 0.8 | UI/UX & Creative Design |

**Data flow:**
```
config.yaml -> persona field ("kamkaew")
                  |
                  v
registry.Get("kamkaew")
    |-> check loaded personas (from .boi/personas/)
    |-> fallback: persona.DefaultPersona()
    |
    v
Persona{Name, Model, Temperature, SystemPrompt}
    |
    v
Injected into agent.Loop builder
```

**Error handling:**
- Persona name not found: falls back to `kamkaew` default
- Persona dir missing: uses embedded defaults from `internal/persona/defaults.go` (compiled into binary)
- YAML parse error: skips bad file, logs warning
- Config missing persona field: uses `kamkaew`

---

### [4] Memory Hook (Phantom DB)

**What happens:**
Before each LLM turn, the Memory Hook searches Phantom DB for relevant past memories using keyword matching. Matching results are formatted as a `<memory-context>` XML block and injected into the system prompt.

**Modules involved:**
| File | Role |
|------|------|
| `internal/memory/hook.go` | `BeforeTurn()` + `AfterTurn()` -- orchestrates memory lifecycle per agent turn |
| `internal/memory/store.go` | Phantom DB -- file-based JSON store. `Open()`, `Save()`, `SearchMemory()`, `InjectMemory()` |
| `internal/memory/prefetch.go` | `Prefetch()` -- keyword search + `InjectMemory()` -- formats memory context block |
| `internal/memory/context.go` | ContextManager -- token budget tracking |
| `internal/memory/extractor.go` | `Extractor` interface + `SimpleExtractor` -- extracts facts from agent output |
| `internal/memory/entity.go` | Weight entity adapter for `MemoryEntry` |

**Data flow:**
```
MemoryHook.BeforeTurn(query)
    |
    v
Store.InjectMemory(query)
    |-> SearchMemory(query, limit=5)  -- keyword match on Key + Content
    |-> sort by score desc
    |-> format as <memory-context> block
    |
    v
Return: "<memory-context>\n1. [solution] login-fix (weight: 0.72)\n   ...\n</memory-context>"
    |
    v
Injected into agent.Loop.Run() before system prompt
```

**Error handling:**
- Memory dir missing (first run): `Open()` creates directory, returns empty store
- Search returns nothing: returns empty string (no context injected)
- Corrupt JSON file: skip file, continue loading others
- Store lock contention: `sync.RWMutex` protects concurrent access

---

### [5] Weight Engine

**What happens:**
The Weight Engine scores memories across 5 dimensions (Truth, Confidence, Importance, Recency, Usage) using a configurable policy. Higher-scored memories are prioritized in prefetch. The engine also supports conflict resolution (ReWeight) between competing claims.

**Modules involved:**
| File | Role |
|------|------|
| `internal/weight/engine.go` | Engine -- `Compute()`, `ComputeAndExplain()`, `ApplyDecay()` |
| `internal/weight/types.go` | `Entity` interface, `Weights` struct, `.Sum()` method |
| `internal/weight/policy.go` | `WeightPolicy` struct, `DefaultPolicy()` |
| `internal/weight/explain.go` | `Explain()` -- generates `WeightExplanation` with per-dimension breakdown |
| `internal/memory/entity.go` | `MemoryEntry` implements `weight.Entity` interface |
| `internal/memory/prefetch.go` | `ReWeight()` -- resolves conflicts between two memories |

**Weight Policy (Default):**
| Dimension | Weight | Description |
|-----------|--------|-------------|
| Truth | 0.35 | Evidence-supported correctness |
| Confidence | 0.25 | Lack of contradictory evidence |
| Usage | 0.20 | Frequency of access |
| Importance | 0.10 | Semantic significance (solution > pattern > fact) |
| Recency | 0.10 | Time decay (1h=1.0, 24h=0.8, 1wk=0.5, older=0.2) |
| DecayRate | 0.05 | Per-hour decay multiplier |

**Formula:**
```
Score = (Truth*0.35 + Confidence*0.25 + Usage*0.20 + Importance*0.10 + Recency*0.10) / sum(weights)
```

**Data flow:**
```
MemoryEntry
    |-> implements weight.Entity interface
    |-> Truth()      -> Score field
    |-> Confidence() -> 0.5 if has content
    |-> Importance() -> 0.9 solution, 0.7 pattern, 0.5 fact
    |-> Recency()    -> hours-based decay
    |-> Usage()      -> 0.5 (static)
    |
    v
Engine.Compute(entity) -> weighted average -> float64 score
    |
    v
Stored in MemoryEntry.Score
Used in SearchMemory() for ordering
```

**Error handling:**
- Nil policy: falls back to `DefaultPolicy()`
- Zero weight sum: returns 0 (no division by zero)
- Decay below floor: clamped at 0.1 minimum

---

### [6] PSC Router (Provider Supply Chain)

**What happens:**
The PSC Router iterates through configured LLM providers in order (PSC_1 through PSC_4). It tries the first provider; if it fails with an HTTP-level error (404, 429, 500, 502, 503, 504, rate limit), it falls back to the next. Non-HTTP errors (auth, config) are not retried.

**Modules involved:**
| File | Role |
|------|------|
| `internal/llm/router.go` | `NewRouter()` -- linear fallback engine. `Complete()` iterates providers |
| `internal/llm/provider.go` | `Provider` interface -- Name(), Complete(), SupportModel() |
| `internal/llm/factory/factory.go` | `LoadProvidersFromEnv()` -- reads PSC_1 through PSC_4 env vars, creates provider instances |
| `internal/llm/providers/openai.go` | OpenAI / OpenAI-compatible provider |
| `internal/llm/providers/anthropic.go` | Anthropic Claude-specific provider |
| `internal/llm/streaming.go` | SSE streaming helper |

**Provider chain:**
```
PSC_1 (primary)     -- try first
    |-> fail (HTTP error)? -> PSC_2
                                |-> fail? -> PSC_3
                                                |-> fail? -> PSC_4
                                                                |-> all fail -> error
```

**Env var format (.env):**
```env
PSC_1_NAME=openai
PSC_1_API_KEY=sk-...
PSC_1_MODEL=gpt-4.1-mini
PSC_1_BASE_URL=      (optional, for proxies/custom endpoints)

PSC_2_NAME=anthropic
PSC_2_API_KEY=sk-ant-...
PSC_2_MODEL=claude-sonnet-5

PSC_3_NAME=openai
PSC_3_BASE_URL=http://localhost:11434/v1
PSC_3_MODEL=llama3.1
```

**Retryable errors (fall through to next provider):**
- `status 404`, `status 429` (rate limit)
- `status 500`, `status 502`, `status 503`, `status 504`
- `internal server error`, `service unavailable`
- `bad gateway`, `gateway timeout`

**Non-retryable errors (immediate failure):**
- Authentication failures (401, 403)
- Invalid request format (400)
- Network connectivity (timeout)

**Data flow:**
```
agent.Loop.callLLM(messages)
    |
    v
llm.NewRouter(providers).Complete(ctx, req)
    |
    for each provider in chain:
        provider.Complete(ctx, req)
        |-> success? -> return response
        |-> HTTP error? -> continue to next
        |-> non-HTTP error? -> return error
    |
    all exhausted? -> return error
```

**Error handling:**
- No providers configured (no .env): falls back to `simulatedResponse()` -- returns mock response
- All providers exhausted: returns "all providers exhausted, last error: ..."
- Environment loading fails: prints warning, continues in simulated mode

---

### [7] LLM Call

**What happens:**
The formatted prompt (system prompt + memory context + user message) is sent to the selected LLM provider. The provider handles API communication and returns a completion response.

**Modules involved:**
| File | Role |
|------|------|
| `internal/agent/loop.go` | `.callLLM()` -- builds request, calls router, handles simulated fallback |
| `internal/llm/provider.go` | `CompletionRequest` / `CompletionResponse` structs |
| `internal/llm/router.go` | Provider selection + fallback |

**Request structure:**
```go
CompletionRequest{
    Messages:    []Message{...},  // system + memory context + user
    MaxTokens:   4096,
    Temperature: p.Temperature,    // from active persona
}
```

**Response structure:**
```go
CompletionResponse{
    Content:      "Generated response text",
    InputTokens:  450,
    OutputTokens: 120,
    Model:        "gpt-4.1-mini",
    Provider:     "openai",
}
```

**Data flow:**
```
buildSystemPrompt() + memory context + user query
    |
    v
CompletionRequest{Messages, MaxTokens, Temperature}
    |
    v
Router.Complete() -> provider.Complete()
    |
    v
CompletionResponse{Content, Tokens, Model, Provider}
```

**Error handling:**
- No providers: simulated response mode
- Provider API error: fallback chain (see [6])
- Timeout: context cancellation propagated
- Empty response: treated as tool request (re-loops)

---

### [8] Agent Loop (ReAct)

**What happens:**
The agent loop implements the ReAct (Reasoning + Acting) pattern. It iterates up to `MaxSteps` (default 15) times. Each iteration calls the LLM, evaluates if the response is a final answer (content > 10 chars) or a tool request, and either returns or continues.

**Modules involved:**
| File | Role |
|------|------|
| `internal/agent/loop.go` | `Loop.Run()` -- main ReAct loop controller |
| `internal/agent/types.go` | `AgentState`, `AgentStep`, `AgentResult` structs |
| `internal/agent/planner.go` | Task decomposition |
| `internal/agent/executor.go` | Tool execution engine |
| `internal/agent/reviewer.go` | Quality review |
| `internal/agent/subagent.go` | Sub-agent delegation |

**Loop state machine:**
```
INIT (thinking)
    |
    v
[Step 1..MaxSteps]
    |-> callLLM(messages)
    |-> check: needsTools(content)?
    |   |-> YES (len < 10): continue loop, append assistant msg
    |   |-> NO:  AfterTurn() -> save memory -> return result
    |
    v
Exceeded MaxSteps? -> return error
```

**State tracking:**
```go
AgentState{
    ID:          "agent_{timestamp}",
    PersonaName: "kamkaew",
    Status:      "thinking" | "done" | "error",
    Task:        "original user query",
    Steps:       []AgentStep{...},
    MemoryUsed:  1,   // count of injected memories
    StartedAt:   time.Now(),
}
```

**Data flow:**
```
Loop.Run(ctx, query)
    |
    v
Init AgentState, BeforeTurn for memory
    |
    v
For i=0..MaxSteps:
    callLLM(ctx, messages)
    |-> success + final answer? -> AfterTurn, return AgentResult
    |-> success + tool req?     -> append to messages, continue
    |-> error?                  -> set status error, return error
    |
    v
Loop exhausted -> "exceeded max steps" error
```

**Error handling:**
- Context cancelled: return immediately with context error
- LLM failure: return "LLM failed at step N: ..."
- Max steps exceeded: return "exceeded max steps (15)"
- Empty response: continue loop (interpreted as tool request)

---

### [9] Post-Processing

**What happens:**
After the agent produces a final response, the Memory Hook extracts facts from the conversation (user query + agent response) and saves them to Phantom DB. Every 10 turns, a nudge is triggered for memory compaction.

**Modules involved:**
| File | Role |
|------|------|
| `internal/memory/hook.go` | `AfterTurn()` -- extract facts, save to store, increment turn counter |
| `internal/memory/extractor.go` | `SimpleExtractor.Extract()` -- extracts facts from conversation (currently pass-through) |
| `internal/memory/store.go` | `Save()` -- writes MemoryEntry as JSON to `.boi/memory/mem_{id}.json` |
| `internal/memory/nudge.go` | Periodic memory maintenance trigger |

**Data flow:**
```
agent.Loop.Run() completes
    |
    v
MemoryHook.AfterTurn(query, response)
    |
    v
Extractor.Extract(ExtractRequest{UserQuery, AgentResponse})
    |-> returns []ExtractedFact{Key, Content, Type, Weight}
    |
    v
For each fact:
    new MemoryEntry{
        MemID:     "mem_{nanosecond}",
        SessionID: "auto",
        Type:      fact.Type,   // "solution", "pattern", "fact"
        Key:       fact.Key,
        Content:   fact.Content,
        Score:     fact.Weight,
        CreatedAt: time.Now(),
    }
    store.Save(entry)
    |
    v
turnCount++
    |-> turnCount % 10 == 0? -> Nudge() (future: compaction)
```

**Error handling:**
- Extractor fails: silent return (don't block agent for memory failure)
- Save fails: error logged, not propagated to user
- Memory dir missing: Open() auto-creates

---

### [10] Response Display

**What happens:**
The agent result is displayed to the user. In CLI mode, it's printed to stdout. In TUI mode, it's added to the chat viewport and rendered in the Bubbletea interface.

**Modules involved:**
| File | Role |
|------|------|
| `internal/cli/agent.go` | Prints `result.Response` to stdout, optionally prints verbose stats |
| `internal/tui/chat.go` | `ChatModel.AddMessage()` -- appends message to viewport |
| `internal/tui/status.go` | Status bar rendering |
| `internal/tui/styles.go` | Lipgloss styles for user/agent/system/error messages |

**CLI output format:**
```
[Agent response text]

---
Steps: 2 | Tokens: 570 | Time: 1.2s
Memories saved: [login-fix]
```

**TUI output format:**
```
> You: debug login bug

<> BOI: Found 3 relevant memories...

    [solution] login-fix   weight: 0.72
      Fixed by URL-encoding password in auth.js line 45

<> BOI: The root cause is special characters in the password field...

[OK] 2 steps  .  450 tokens  .  0.3s
```

**Error handling:**
- Response is empty: display "No response generated"
- Verbose mode: show step count, token usage, execution time
- TUI window too small: text truncated gracefully by viewport

---

## Sub-Flow A: TUI Flow

```
boi (no args)
    |
    v
main.go: len(os.Args) <= 1 -> runTUI()
    |
    v
+-------------------------------------------------------------+
|  TUI FLOW                                                    |
+-------------------------------------------------------------+
|                                                              |
|  [A1] Splash Screen (SplashModel)                            |
|       Load logo (boiLogoDOS), detect workspace, count        |
|       personas/providers/memories/skills                     |
|       |-> "Press Enter to start..."                          |
|       |                                                       |
|       v                                                       |
|  [A2] Chat Interface (ChatModel)                             |
|       Bubbletea viewport with message rendering               |
|       Input area at bottom with key bindings                 |
|       |                                                       |
|       v                                                       |
|  [A3] Event Loop (Bubbletea)                                 |
|       |-> KeyMsg -> route to handler                         |
|       |   Tab    -> switch persona                           |
|       |   Enter  -> send message to agent loop               |
|       |   Ctrl+N -> newline in input                         |
|       |   Ctrl+Q -> quit                                     |
|       |                                                       |
|       |-> agent.Loop.Run() returns -> AddMessage to chat     |
|       |                                                       |
|       v                                                       |
|  [A4] Persona Switching                                      |
|       Tab key cycles through loaded personas                 |
|       Updates status bar with new persona name               |
|       Next message uses new persona's system prompt          |
|                                                              |
+-------------------------------------------------------------+

Key files: cmd/boi/tui.go, internal/tui/{chat,splash,status,input,help,styles,eightart}.go
```

**Error handling:**
- Terminal too small: minimum 80x20 viewport
- No personas loaded: shows only "kamkaew" default
- Agent error: displayed as error message in chat

---

## Sub-Flow B: Memory Flow

```
Conversation (query + response)
    |
    v
+-------------------------------------------------------------+
|  MEMORY FLOW                                                 |
+-------------------------------------------------------------+
|                                                              |
|  [B1] EXTRACT                                               |
|       MemoryHook.AfterTurn(query, response)                  |
|       SimpleExtractor.Extract() -> []ExtractedFact           |
|       |-> future: LLM-based extraction for richer facts     |
|       |                                                       |
|       v                                                       |
|  [B2] SAVE                                                  |
|       Each fact -> MemoryEntry{ID, Type, Key, Content, Score}|
|       Store.Save() -> writes JSON to .boi/memory/            |
|       |-> mem_{nanosecond}.json                              |
|       |                                                       |
|       v                                                       |
|  [B3] SEARCH (on next query)                                |
|       MemoryHook.BeforeTurn(newQuery)                         |
|       Store.InjectMemory(query) -> keyword match             |
|       |-> Key contains word? -> +3.0 score                  |
|       |-> Content contains word? -> +1.0 score               |
|       |-> sort by score desc, limit 5                       |
|       |                                                       |
|       v                                                       |
|  [B4] INJECT                                                |
|       Format as <memory-context> XML block                    |
|       Inserted into system prompt before user message        |
|       |                                                       |
|       v                                                       |
|  [B5] MAINTAIN (every 10 turns)                              |
|       Nudge() - future: compaction, TTL cleanup              |
|       Decay score by time (ApplyDecay)                       |
|                                                              |
+-------------------------------------------------------------+

Storage format (.boi/memory/mem_{id}.json):
{
  "id": "mem_1785403325978479400",
  "session_id": "auto",
  "type": "solution",
  "key": "login-fix",
  "content": "Fixed by URL-encoding password in auth.js line 45",
  "score": 0.72,
  "created_at": "2026-07-30T...",
  "ttl": 0
}
```

**Error handling:**
- Store directory missing: `Open()` creates it
- JSON marshal error: logged, entry dropped
- Read error during search: file skipped
- Old entries (TTL expired): cleaned by `CleanExpired()`

---

## Sub-Flow C: Weight Flow

```
New memory fact created
    |
    v
+-------------------------------------------------------------+
|  WEIGHT FLOW                                                 |
+-------------------------------------------------------------+
|                                                              |
|  [C1] INITIAL SCORE                                         |
|       MemoryHook assigns base score from ExtractedFact.Weight|
|       Default: 1.0 if not specified                          |
|       |                                                       |
|       v                                                       |
|  [C2] DIMENSION COMPUTATION                                 |
|       MemoryEntry implements weight.Entity:                  |
|       |-> Truth()      = entry.Score                         |
|       |-> Confidence() = 0.5 (static baseline)               |
|       |-> Importance() = 0.9 solution / 0.7 pattern / 0.5 fact|
|       |-> Recency()    = 1.0 (<1h) / 0.8 (<24h) / 0.5 (<1wk)|
|       |-> Usage()      = 0.5 (static)                        |
|       |                                                       |
|       v                                                       |
|  [C3] POLICY WEIGHTING                                      |
|       WeightPolicy{                                          |
|         Truth: 0.35, Confidence: 0.25,                       |
|         Usage: 0.20, Importance: 0.10, Recency: 0.10         |
|       }                                                      |
|       Score = weighted average across 5 dimensions           |
|       |                                                       |
|       v                                                       |
|  [C4] CONFLICT CHECK                                        |
|       New fact contradicts existing?                         |
|       |-> YES -> Store.ReWeight(oldID, newID, evidence)      |
|       |   new evidence -> newClaim +0.10, oldClaim -0.10     |
|       |   no evidence  -> higher score +0.05, lower -0.05    |
|       |   floor at 0.1                                       |
|       |                                                       |
|       v                                                       |
|  [C5] TIME DECAY                                            |
|       Every hour: ApplyDecay(score, hours, 0.05)             |
|       score = score - hours * decayRate                      |
|       floor at 0.1                                           |
|       |                                                       |
|       v                                                       |
|  [C6] EXPLAIN                                               |
|       boi weight explain <id>                                |
|       WeightExplanation{                                     |
|         Breakdown: [                                        |
|           {dim:"truth", raw:0.85, policy:0.35, contrib:0.30, |
|            reason:"strong evidence"},                        |
|           {dim:"confidence", raw:0.50, policy:0.25, contrib:0.13,|
|            reason:"no conflicts"},                           |
|           ...                                                |
|         ],                                                   |
|         FinalScore: 0.72                                     |
|       }                                                      |
|                                                              |
+-------------------------------------------------------------+

Example: boi weight explain mem_1785403325978479400
    +Truth       : strong evidence
    +Confidence  : no conflicts
    +Importance  : importance score
    +Recency     : seen today
    +Usage       : usage frequency
    Final: 0.72
```

**Error handling:**
- Memory not found for explain: "one or both memories not found"
- ReWeight with non-existent IDs: error returned
- Policy sum = 0: division guarded, returns 0

---

## Complete End-to-End Data Flow

```
boi ask "debug login bug" -p dang -v
    |
    v
[1] CLI Parser
    command="ask", query="debug login bug"
    persona="dang", steps=15, verbose=true
    |
    v
[2] Workspace Detection
    root=F:\.Agent-CLI\BOI-CLI
    .boi/ -> config.yaml + personas/ + skills/ + memory/
    |
    v
[3] Persona Loader
    Active: dang (Debug Specialist)
    Model: gpt-4.1-mini, Temp: 0.2
    SystemPrompt: "You are a debugging specialist..."
    |
    v
[4] Memory Hook
    BeforeTurn("debug login bug")
    InjectMemory searches Phantom DB for "debug login bug"
    |-> found 2 matches
    |-> formatted as <memory-context> block
    |
    v
[5] Weight Engine
    Each memory scored by DefaultPolicy
    Ranked by score desc
    Top 5 injected
    |
    v
[6] PSC Router
    Factory.LoadProvidersFromEnv()
    |-> PSC_1: openai/gpt-4.1-mini (primary)
    |-> PSC_2: anthropic/claude-sonnet-5 (fallback)
    Router created with [openai, anthropic]
    |
    v
[7] LLM Call
    Messages: [system prompt + memory context, user query]
    Router.Complete() -> openai.Complete()
    Response: {content: "...", tokens: {in:450, out:120}}
    |
    v
[8] Agent Loop (step 1)
    callLLM returns -> content length > 10 -> final answer
    AfterTurn() executed
    |
    v
[9] Post-Processing
    Extract facts from conversation
    Save new memory entries to Phantom DB
    |
    v
[10] Response Display
    Print response content
    [verbose] Steps: 1 | Tokens: 570 | Time: 1.2s
```

---

*End of SYSTEM_FLOW.md*
