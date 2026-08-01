# 🏛️ BOI CLI — แผนแม่บท

> สร้างเมื่อ: 27 กรกฎาคม 2026
> โดย: คำปัน (Kampun) — BOI Family
> สถานะ: กำลังวางแผน (PRE-BUILD)

---

## 📋 สารบัญ

1. [Identity & Branding](#1-identity--branding)
2. [Chimera Architecture — DNA Sources](#2-chimera-architecture--dna-sources)
3. [8-Layer Architecture](#3-8-layer-architecture)
4. [Tech Stack](#4-tech-stack)
5. [Greek God Level System](#5-greek-god-level-system)
6. [10 Evidence Dimensions](#6-10-evidence-dimensions)
7. [Phase Plan — 6 Phases](#7-phase-plan--6-phases)
8. [Project Structure](#8-project-structure)
9. [Evolution System](#9-evolution-system)

---

## 1. Identity & Branding

```
BOI (Brand)
 │
 └── BOI CLI (Runtime / Agent)
      ├── Kampun    → Learning Loop (GEPA Evolution)
      ├── PSC       → Provider Supply Chain (6+ LLMs)
      ├── Phantom DB→ Memory & Context
      ├── Persona   → 6 Agent Profiles
      └── MCP Tools → Extension Interface
```

| Level | ชื่อ | บทบาท |
|-------|------|--------|
| **Brand** | BOI | แบรนด์หลักของระบบ |
| **Runtime** | BOI CLI | CLI Agent ที่ทำงานจริง |
| **Learning** | Kampun | วงจรเรียนรู้ + GEPA Evolution |

---

## 2. Chimera Architecture — DNA Sources

> คิเมร่า = ดึง DNA/Pattern จากหลาย CLI มาประกอบเป็นสถาปัตยกรรมใหม่
> ไม่ใช่การ fork หรือ merge source code โดยตรง

```
OpenCode    → Agent System + Permission + 9-Layer Edit ──┐
Hermes      → GEPA Evolution + Multi-Terminal ────────────┤
Claude CLI  → Memory (CLAUDE.md) + Extended Thinking ─────┤
Codex CLI   → Sandbox Modes + Approval Policy ────────────┼──→ BOI CLI
Antigravity → Multi-Provider Router + 8-Model Support ────┤
Agent Zero  → Computer-as-Tool + Adaptive Memory ─────────┤
ZeroClaw    → Trait-Based Subsystems + WASM + Security ───┘
                               ↓
                    Pattern / Architecture / UX / Workflow
                    (ไม่เอา Source Code)
```

### สิ่งที่เอามาจากแต่ละตัว

| # | Agent | Core Pattern | สิ่งที่เอามาใช้ |
|---|-------|-------------|----------------|
| 1 | **Hermes** | Self-Evolution Loop | GEPA Engine, DSPy prompt evolution |
| 2 | **OpenCode** | Agent System + Event Bus | Permission Rulesets, 9-Layer Edit Fallback |
| 3 | **Claude Code** | Extended Thinking + Memory | CLAUDE.md hierarchy, Context Compaction |
| 4 | **Codex CLI** | Two-Axis Autonomy | 3 Sandbox Modes, Approval Policy |
| 5 | **Antigravity** | Multi-Provider Auto-Routing | 8-Model Router, Task Classification |
| 6 | **Agent Zero** | Computer-as-Tool | Adaptive Memory, State Machine Workflow |
| 7 | **ZeroClaw** | Ultra-Lightweight Runtime | Trait-Based Subsystems, Security-First |

---

## 3. 8-Layer Architecture

```
┌─────────────────────────────────────────────────────────────────────────────┐
│  L1: CLI & INTERFACE        Cobra CLI + Flags + Help                       │
│  L2: TUI (Optional)         Bubbletea (กันไว้เผื่อใช้)                       │
│  L3: ORCHESTRATION CORE     ReAct Loop + Planner + Self-Healing             │
│  L4: PERSONA SYSTEM         6 Personas (BOI/Kamkaew/Kampun/Dang/Don/Kine)   │
│  L5: SKILL SYSTEM           Skill Runtime + Loader + Registry + MCP         │
│  L6: PROVIDER SUPPLY CHAIN  OpenAI/Anthropic/Google/xAI/DeepSeek/Ollama     │
│  L7: MEMORY & PHANTOM DB    SQLite + FTS5 + Context + Cross-Session        │
│  L8: SELF-EVOLUTION ENGINE  GEPA + Trace Collector + Level Manager         │
└─────────────────────────────────────────────────────────────────────────────┘
```

---

## 4. Tech Stack

| Component | เทคโนโลยี | เหตุผล |
|-----------|----------|--------|
| **Language** | Go 1.24+ | Single binary, เร็ว, memory ต่ำ, concurrency เยี่ยม |
| **CLI Parser** | Cobra + Viper | มาตรฐาน Go CLI, config hot-reload |
| **TUI (optional)** | Bubbletea | Elm architecture, กันไว้เผื่อใช้ |
| **Database** | modernc.org/sqlite | Zero-dependency SQLite |
| **Vector Search** | SQLite FTS5 | ไม่ต้อง external vector DB |
| **MCP** | mark3labs/mcp-go | Official SDK, zero-dependency |
| **LLM Router** | Custom Go | Full control over routing |
| **Sandbox** | wazero (WASM) | Lightweight isolation |
| **Config** | YAML (viper) | Human-readable, hot-reload |
| **Logging** | slog (stdlib) | Zero external dep for logging |

### Python ใช้เฉพาะ

```
AI / Vision / Training / Research เท่านั้น
Runtime หลัก = Go ล้วน
```

---

## 5. Greek God Level System

```
─────────────────────────────────────────────────────────────────────
 LEVEL     GOD              DOMAIN              AI CAPABILITY
─────────────────────────────────────────────────────────────────────
 Lv9   🌌 Aether        แสงสวรรค์, อวกาศ       Transcendence
 Lv8   ⚡ Zeus          จ้าวโอลิมปัส            System Governance
 Lv7   🏹 Artemis       นายพรานอิสระ           Autonomous Operation
 Lv6   ☀️ Apollo        ศิลปะ, คำทำนาย         Mastery & Creation
 Lv5   🦉 Athena        ปัญญา, ยุทธศาสตร์       Strategic Wisdom
 Lv4   👟 Hermes        สื่อสาร, ความเร็ว        Speed & Connection
 Lv3   🔨 Hephaestus    ช่างเหล็ก, งานฝีมือ     Crafting & Building
 Lv2   🔥 Prometheus    ไฟ, การเรียนรู้          Learning & Discovery
 Lv1   ⏳ Cronus        กาลเวลา, พื้นฐาน         Foundation & Awareness
─────────────────────────────────────────────────────────────────────
```

### Mapping กับ Development Phase

| Phase | Level | God | Achievement |
|-------|-------|-----|-------------|
| Phase 1 | Lv1 ⏳ Cronus | Foundation | CLI ทำงานได้ |
| Phase 2 | Lv2 🔥 Prometheus | Fire-Giver | ทุก Provider ยิงได้ |
| Phase 3 | Lv3 🔨 Hephaestus | Craftsman | Persona ครบ 6 ตัว |
| Phase 4 | Lv4 👟 Hermes | Messenger | Skill System ทำงาน |
| Phase 5 | Lv5 🦉 Athena | Wisdom | Memory + Context |
| Phase 6 | Lv6 ☀️ Apollo | Mastery | Agent ทำงานได้ |
| Post-v1 | Lv7 🏹 Artemis | Independence | Autonomous Mode |
| Post-v1 | Lv8 ⚡ Zeus | Governance | Multi-Agent Orchestration |
| Post-v1 | Lv9 🌌 Aether | Transcend | Self-Evolution Complete |

### แต่ละ Level มี 3 Tiers

```
Tier            ช่วงคะแนน        ความหมาย
────────────────────────────────────────────────
🔸 Low         0.0 - 3.9      ก้าวแรกของระดับนั้น
🔹 Mid         4.0 - 6.9      ทรงตัวได้ในระดับ
💎 High        7.0 - 10.0     พร้อมเลื่อนขึ้นระดับต่อไป
```

---

## 6. 10 Evidence Dimensions

| Code | Dimension | นิยาม | Lv1-3 Wt | Lv4-6 Wt | Lv7-9 Wt |
|------|-----------|-------|----------|----------|----------|
| **ACC** | Accuracy | ความถูกต้องของ answer/tool | 0.25 | 0.10 | 0.05 |
| **AUT** | Autonomy | ทำงานได้เองโดยไม่ต้องถาม | 0.10 | 0.15 | 0.10 |
| **CTX** | Context | เข้าใจ project/user/business | 0.10 | 0.10 | 0.10 |
| **EFF** | Efficiency | ใช้ token น้อยแต่ได้ผล | 0.05 | 0.10 | 0.15 |
| **LRN** | Learning | เรียนรู้จากความผิดพลาด | 0.05 | 0.15 | 0.15 |
| **COL** | Collaboration | ทำงานกับ user/agent อื่น | 0.10 | 0.05 | 0.05 |
| **CON** | Consistency | ทำได้สม่ำเสมอไม่เพี้ยน | 0.10 | 0.10 | 0.05 |
| **ADP** | Adaptability | ปรับตัวกับ stack/project ใหม่ | 0.05 | 0.10 | 0.15 |
| **SEC** | Security | ไม่ leak secrets, safe actions | 0.15 | 0.10 | 0.10 |
| **SPD** | Speed | Response time, time-to-task | 0.05 | 0.05 | 0.10 |

### สูตรคำนวณ

```
Level Score = Σ(dimension_score × level_weight) / Σ(weights)

เลื่อน Tier:   Score ≥ 6.0 → ถัดไป (ภายในระดับเดียวกัน)
เลื่อน Level:  Score ≥ 7.0 + เงื่อนไขเฉพาะระดับ
Regression:    Score < 4.0 → ตก 1 Tier
               Score < 2.0 → ตก 1 Level
```

---

## 7. Phase Plan — 6 Phases

### Phase 1: Core Runtime (2 weeks)

**เป้าหมาย:** CLI ที่เซตอัปแล้ววิ่งได้ — Config, Workspace, Logger, Command

**Week 1 — Structure + CLI Shell**

| Task | File | Description |
|------|------|-------------|
| 1.1 | `cmd/boi/main.go` | CLI entry point |
| 1.2 | `internal/cli/root.go` | `boi` base command |
| 1.3 | `internal/cli/run.go` | `boi run "..."` |
| 1.4 | `internal/cli/init.go` | `boi init` — create `.boi/` |
| 1.5 | `internal/cli/config.go` | `boi config` — show/edit config |
| 1.6 | `go.mod` + `Makefile` | Go module + build targets |

**Week 2 — Config + Workspace + Logger + Command**

| Task | File | Description |
|------|------|-------------|
| 2.1 | `internal/config/config.go` | Load `.boi/config.yaml` |
| 2.2 | `internal/config/defaults.go` | Default values |
| 2.3 | `internal/config/watcher.go` | Hot-reload on file change |
| 2.4 | `internal/workspace/workspace.go` | Detect project root |
| 2.5 | `internal/workspace/scanner.go` | Scan project structure |
| 2.6 | `internal/logger/logger.go` | Structured logging (slog) |
| 2.7 | `internal/logger/levels.go` | DEBUG/INFO/WARN/ERROR |
| 2.8 | `internal/command/executor.go` | Shell command execution |
| 2.9 | `internal/command/sandbox.go` | Basic allow/deny list |

**✅ Deliverable:** `boi init` → สร้าง `.boi/` + `boi run "ls -la"` → รัน shell ได้

---

### Phase 2: PSC — Provider Supply Chain (2 weeks)

**เป้าหมาย:** ยิงทุก LLM Provider ได้ — หัวใจของ BOI CLI

**Week 3 — Provider Interface + 2 Providers**

| Task | File | Description |
|------|------|-------------|
| 3.1 | `internal/llm/provider.go` | Provider interface definition |
| 3.2 | `internal/llm/streaming.go` | SSE streaming handler |
| 3.3 | `internal/llm/providers/openai.go` | OpenAI / GPT series |
| 3.4 | `internal/llm/providers/anthropic.go` | Anthropic / Claude |
| 3.5 | `internal/llm/cost.go` | Cost tracking struct |
| 3.6 | LLM integration test | Test with real API keys |

**Week 4 — 4 More Providers + Router**

| Task | File | Description |
|------|------|-------------|
| 4.1 | `internal/llm/providers/gemini.go` | Google Gemini |
| 4.2 | `internal/llm/providers/grok.go` | xAI Grok |
| 4.3 | `internal/llm/providers/deepseek.go` | DeepSeek |
| 4.4 | `internal/llm/providers/ollama.go` | Local models |
| 4.5 | `internal/llm/router.go` | Provider router |
| 4.6 | `internal/llm/cache.go` | Response caching |

**✅ Deliverable:** `boi run "hello"` → เลือก provider จาก config → สตรีม response

---

### Phase 3: Persona (2 weeks)

**เป้าหมาย:** 6 Personas = 6 Agent Config Profiles

**Week 5 — Persona System + 2 Personas**

| Task | File | Description |
|------|------|-------------|
| 5.1 | `internal/persona/registry.go` | Persona registry |
| 5.2 | `internal/persona/loader.go` | Load `.boi/personas/*.yaml` |
| 5.3 | `boi/personas/kamkaew.yaml` | Kamkaew — Runtime Agent |
| 5.4 | `boi/personas/kampun.yaml` | Kampun — Learning Loop |

**Week 6 — 4 More Personas + Switching**

| Task | File | Description |
|------|------|-------------|
| 6.1 | `boi/personas/boi.yaml` | BOI — Default Orchestrator |
| 6.2 | `boi/personas/dang.yaml` | Dang — Debug Specialist |
| 6.3 | `boi/personas/don.yaml` | Don — Documentation |
| 6.4 | `boi/personas/kine.yaml` | Kine — Creative Design |
| 6.5 | `internal/cli/persona.go` | `boi persona list/switch` |
| 6.6 | Persona switching integration | CLI arg + config-based |

**✅ Deliverable:** `boi persona switch kamkaew` + `boi run "..."` → response ตามบุคลิก

---

### Phase 4: Skill (2 weeks)

**เป้าหมาย:** Dynamic skill system — โหลด SKILL.md runtime

**Week 7 — Skill Runtime + Loader**

| Task | File | Description |
|------|------|-------------|
| 7.1 | `internal/skill/runtime.go` | Skill execution engine |
| 7.2 | `internal/skill/loader.go` | Load SKILL.md files |
| 7.3 | `internal/skill/registry.go` | Skill registry |
| 7.4 | `internal/skill/parser.go` | Parse markdown → tool def |

**Week 8 — Built-in Skills + Skill Commands**

| Task | File | Description |
|------|------|-------------|
| 8.1 | `internal/skill/builtins/git.skill.md` | Git operations skill |
| 8.2 | `internal/skill/builtins/web.skill.md` | Web search/fetch skill |
| 8.3 | `internal/cli/skill.go` | `boi skill list/load/create` |
| 8.4 | `internal/mcp/client.go` | MCP client integration |

**✅ Deliverable:** `boi skill load git.skill.md` + เรียกใช้ skill ผ่าน Kamkaew

---

### Phase 5: Memory & Phantom DB (2 weeks)

**เป้าหมาย:** Cross-session memory + context management

**Week 9 — Phantom DB Core**

| Task | File | Description |
|------|------|-------------|
| 9.1 | `internal/memory/phantom_db.go` | SQLite-backed memory store |
| 9.2 | `internal/memory/history.go` | Session history |
| 9.3 | `internal/memory/context.go` | Context window manager |
| 9.4 | `internal/memory/compaction.go` | Token budget compaction |

**Week 10 — Vector Memory + Cross-Session**

| Task | File | Description |
|------|------|-------------|
| 10.1 | `internal/memory/vector.go` | FTS5 + embedding search |
| 10.2 | `internal/memory/cross_session.go` | Cross-session memory |
| 10.3 | `internal/memory/repomap.go` | Codebase understanding |
| 10.4 | `internal/memory/claudemd.go` | CLAUDE.md hierarchy |
| 10.5 | `internal/cli/memory.go` | `boi memory show/search` |

**✅ Deliverable:** จำ context ข้าม session, ค้นหาความรู้เดิม, inject memory ก่อนทุก turn

---

### Phase 6: Agent (3 weeks)

**เป้าหมาย:** Agent เกิดหลังทุกอย่าง — ตอนนี้ Runtime, PSC, Persona, Skill, Memory พร้อม

**Week 11 — Planner + Executor**

| Task | File | Description |
|------|------|-------------|
| 11.1 | `internal/agent/planner.go` | Task decomposition |
| 11.2 | `internal/agent/executor.go` | Tool execution engine |
| 11.3 | `internal/agent/loop.go` | ReAct loop |

**Week 12 — Reviewer + Sub-Agent**

| Task | File | Description |
|------|------|-------------|
| 12.1 | `internal/agent/reviewer.go` | Quality review |
| 12.2 | `internal/agent/sub_agent.go` | Sub-agent delegation |
| 12.3 | `internal/agent/coordinator.go` | Multi-agent orchestration |

**Week 13 — Evolution Engine + Polish**

| Task | File | Description |
|------|------|-------------|
| 13.1 | `internal/evolution/trace.go` | Trace collector |
| 13.2 | `internal/evolution/gepa.go` | GEPA evolution engine |
| 13.3 | `internal/evolution/pattern.go` | Pattern detector |
| 13.4 | `internal/evolution/level.go` | Level manager |
| 13.5 | `internal/evolution/score.go` | 10-dimension scoring |
| 13.6 | `internal/cli/evolve.go` | `boi evolve` command |
| 13.7 | `.evolution/config.toml` | Evolution config |

**✅ Deliverable:** Agent ทำงานครบวงจร + Self-Evolution Engine

---

## 8. Project Structure

```
boi-cli/
├── cmd/
│   └── boi/
│       └── main.go                    # CLI entry point
├── internal/
│   ├── cli/                           # CLI commands (Cobra)
│   │   ├── root.go                    # base command
│   │   ├── run.go                     # boi run
│   │   ├── init.go                    # boi init
│   │   ├── config.go                  # boi config
│   │   ├── persona.go                 # boi persona
│   │   ├── skill.go                   # boi skill
│   │   ├── memory.go                  # boi memory
│   │   └── evolve.go                  # boi evolve
│   ├── config/                        # Config system
│   │   ├── config.go                  # Load/save config
│   │   ├── defaults.go                # Default values
│   │   └── watcher.go                 # Hot-reload
│   ├── workspace/                     # Workspace management
│   │   ├── workspace.go               # Project root detection
│   │   └── scanner.go                 # Project scanning
│   ├── logger/                        # Logging system
│   │   ├── logger.go                  # Structured logging
│   │   └── levels.go                  # Log levels
│   ├── command/                       # Command execution
│   │   ├── executor.go                # Shell execution
│   │   └── sandbox.go                 # Allow/deny sandbox
│   ├── llm/                           # Provider Supply Chain
│   │   ├── provider.go                # Provider interface
│   │   ├── router.go                  # Provider router
│   │   ├── streaming.go               # SSE handler
│   │   ├── cost.go                    # Cost tracking
│   │   ├── cache.go                   # Response cache
│   │   └── providers/
│   │       ├── openai.go              # OpenAI
│   │       ├── anthropic.go           # Anthropic
│   │       ├── gemini.go              # Google
│   │       ├── grok.go                # xAI
│   │       ├── deepseek.go            # DeepSeek
│   │       └── ollama.go              # Local
│   ├── persona/                       # Persona system
│   │   ├── registry.go                # Persona registry
│   │   └── loader.go                  # YAML loader
│   ├── skill/                         # Skill system
│   │   ├── runtime.go                 # Skill execution
│   │   ├── loader.go                  # SKILL.md loader
│   │   ├── registry.go                # Skill registry
│   │   └── builtins/                  # Built-in skills
│   ├── mcp/                           # MCP protocol
│   │   └── client.go                  # MCP client
│   ├── memory/                        # Phantom DB
│   │   ├── phantom_db.go              # SQLite store
│   │   ├── history.go                 # Session history
│   │   ├── context.go                 # Context manager
│   │   ├── compaction.go              # Token compaction
│   │   ├── vector.go                  # FTS5 search
│   │   ├── cross_session.go           # Cross-session
│   │   ├── repomap.go                 # Codebase map
│   │   └── claudemd.go                # CLAUDE.md loader
│   ├── agent/                         # Agent core
│   │   ├── loop.go                    # ReAct loop
│   │   ├── planner.go                 # Task decomposition
│   │   ├── executor.go                # Tool execution
│   │   ├── reviewer.go                # Quality review
│   │   ├── sub_agent.go               # Sub-agent delegation
│   │   └── coordinator.go             # Multi-agent orchestration
│   └── evolution/                     # Self-evolution
│       ├── trace.go                   # Trace collector
│       ├── gepa.go                    # GEPA engine
│       ├── pattern.go                 # Pattern detector
│       ├── level.go                   # Level manager
│       └── score.go                   # 10-dimension score
├── pkg/
│   └── types/
│       └── types.go                   # Shared types
├── .boi/                              # User's project config
│   ├── config.yaml                    # Main config
│   ├── permissions.yaml               # Permission rules
│   └── personas/                      # Persona definitions
│       ├── boi.yaml
│       ├── kamkaew.yaml
│       ├── kampun.yaml
│       ├── dang.yaml
│       ├── don.yaml
│       └── kine.yaml
├── .evolution/                        # Evolution data
│   ├── config.toml                    # Evolution config
│   ├── evidence.db                    # SQLite evidence
│   ├── scores.json                    # Current scores
│   └── history.json                   # Level history
├── docs/
│   ├── README.md
│   ├── ARCHITECTURE.md
│   ├── COMMANDS.md
│   └── EVOLUTION.md
├── scripts/
│   ├── build.sh
│   └── release.sh
├── go.mod
├── go.sum
├── Makefile
└── README.md
```

---

## 9. Evolution System

### Circle of Learning (ทุก session)

```
  ┌─────────────────────────────────────────────────────────┐
  │              EVERY SESSION CYCLE                         │
  │                                                         │
  │  1. EXECUTE ──→ 2. COLLECT ──→ 3. SCORE ──→ 4. EVALUATE │
  │       ↑                                      │          │
  │      │                                      ▼          │
  │  8. FEEDBACK ← 7. LEVEL CHECK ← 6. EVOLVE ← 5. ANALYZE │
  │                                                         │
  └─────────────────────────────────────────────────────────┘
```

### Level Progression

```
Cronus (Phase 1)    → Foundation CLI
Prometheus (Phase 2)→ Provider Access
Hephaestus (Phase 3)→ Persona Mastery
Hermes (Phase 4)    → Skill Connection
Athena (Phase 5)    → Knowledge & Memory
Apollo (Phase 6)    → Agent Mastery
Artemis (Post-v1)   → Autonomous Operation
Zeus (Post-v1)      → System Governance
Aether (Post-v1)    → Transcendence
```

### ระดับปัจจุบัน

```
BOI CLI — ยังไม่เริ่ม BUILD
Level: ⏳ Cronus — 🔸 Low
Score: 0.0
Next Phase: Phase 1 — Core Runtime
```

---

## 📅 Timeline สรุป

```
PHASE 1: Core Runtime     ████████░░░░  Week 1-2
PHASE 2: PSC              ████████░░░░  Week 3-4
PHASE 3: Persona          ████████░░░░  Week 5-6
PHASE 4: Skill            ████████░░░░  Week 7-8
PHASE 5: Memory           ████████░░░░  Week 9-10
PHASE 6: Agent            ████████████  Week 11-13
─────────────────────────────────────────────
Total: 13 weeks → BOI CLI v1.0
```

---

## 🔗 References

- เอกสารภาษาไทย: `หลักการสร้าง Agent.md` (อยู่ในโฟลเดอร์นี้)
- Skill: `expert-god-of-agent/SKILL.md`
- Skill: `expert-god-of-builder/SKILL.md`
- OpenCode Architecture: `github.com/anomalyco/opencode`
- MCP Protocol: `modelcontextprotocol.io`
- Bubbletea: `github.com/charmbracelet/bubbletea`

---

*แผนแม่บทนี้ปรับปรุงล่าสุด: 27 กรกฎาคม 2026*
*โดย: คำปัน (Kampun) — BOI Family*
*อนุมัติโดย: หัวหน้าเอ (Bro)*
