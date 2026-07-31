# <img src="data:image/svg+xml;base64,PHN2ZyB3aWR0aD0iNDAiIGhlaWdodD0iNDAiIHZpZXdCb3g9IjAgMCA0MCA0MCIgZmlsbD0ibm9uZSIgeG1sbnM9Imh0dHA6Ly93d3cudzMub3JnLzIwMDAvc3ZnIj4KICA8cmVjdCB3aWR0aD0iNDAiIGhlaWdodD0iNDAiIHJ4PSI4IiBmaWxsPSIjNkM2M0ZGIi8+CiAgPHRleHQgeD0iMjAiIHk9IjI4IiB0ZXh0LWFuY2hvcj0ibWlkZGxlIiBmb250LXNpemU9IjIyIiBmb250LWZhbWlseT0ibW9ub3NwYWNlIiBmaWxsPSIjRkZGIj5CT0k8L3RleHQ+Cjwvc3ZnPg==" width="40" height="40" align="left" alt="BOI"> BOI CLI

> 🧬 **Chimera Architecture** — DNA from OpenCode, Hermes, Claude Code, Codex CLI, Antigravity, Agent Zero, ZeroClaw
>
> 🚀 The BOI Family's AI Agent Runtime — **Built in Go. Runs in Terminal. Thinks Like You.**

<p align="center">
  <img src="https://img.shields.io/badge/Go-1.26-00ADD8?style=flat&logo=go" alt="Go">
  <img src="https://img.shields.io/badge/Version-1.0.0-6C63FF?style=flat" alt="Version">
  <img src="https://img.shields.io/badge/Binary-7.5MB-success?style=flat" alt="Binary">
  <img src="https://img.shields.io/badge/License-MIT-blue?style=flat" alt="License">
  <img src="https://img.shields.io/badge/Platform-Windows%20%7C%20macOS%20%7C%20Linux-lightgrey?style=flat" alt="Platform">
</p>

---

<p align="center">
  <b>BOI CLI</b> ไม่ใช่แค่ terminal tool — มันคือ <b>Agent Runtime ที่มีชีวิต</b><br>
  เรียนรู้จากทุก conversation • จดจำสิ่งที่สำคัญ • พัฒนาตัวเองทุก session
</p>

---

## 📸 TUI Preview

```
┌─ BOI CLI — 🔨 Hephaestus | Mid ── Persona: kampun ─── idle ────────┐
│                   █████████████████████████████████████              │
├─────────────────────────────────────────────────────────────────────┤
│                                                                     │
│  ▶ You: login bug เกิดจากอะไร                                       │
│                                                                     │
│  ◆ BOI: 🔍 Phantom DB พบ 3 ความจำที่เกี่ยวข้อง...                    │
│                                                                     │
│  ┌─ <memory-context> ───────────────────────────────────────────┐  │
│  │ [solution] login-fix   weight: 0.72                          │  │
│  │   Fixed by URL-encoding password in auth.js line 45          │  │
│  │ [fact]     login-bug   weight: 0.65                          │  │
│  │   Login page crashes when password has special characters    │  │
│  │ [fact]     deploy-st   weight: 0.60                          │  │
│  │   v1.2.3 deployed — no issues reported                       │  │
│  └──────────────────────────────────────────────────────────────┘  │
│                                                                     │
│  ◆ BOI: จาก Phantom DB — สาเหตุคือ special characters ใน password    │
│  ที่ไม่ได้ถูก URL-encode ตอนส่งไป server                             │
│                                                                     │
│  ✅ 2 steps  ·  450 tokens  ·  0.3s                                 │
│                                                                     │
├─────────────────────────────────────────────────────────────────────┤
│ > login bug มันเกิดจากอะไรครับ                                       │
├─────────────────────────────────────────────────────────────────────┤
│  Tab:persona  Enter:send  Ctrl+N:newline  Ctrl+Q:quit               │
└─────────────────────────────────────────────────────────────────────┘
```

---

## ⚡ Quick Start

```bash
# Clone & build
git clone https://github.com/wersoul-source/BOI-CLI.git
cd BOI-CLI
go build -o bin/boi.exe ./cmd/boi

# Initialize workspace
./bin/boi init

# Setup AI (optional)
cp .env.example .env
# Edit .env → add PSC_1_NAME, PSC_1_API_KEY, PSC_1_MODEL

# Launch TUI
./bin/boi

# Or use CLI mode
./bin/boi ask "อธิบายโค้ดนี้"
./bin/boi run "git status"
```

---

## 🏗️ Architecture

```
                    ┌──────────────────────────┐
                    │    🖥️  TUI (Bubbletea)    │  ← Full-screen terminal UI
                    │    Tab:persona /:commands │
                    └────────────┬─────────────┘
                                 │
          ┌──────────────────────┼──────────────────────┐
          │                      │                      │
    ┌─────▼─────┐         ┌──────▼──────┐        ┌─────▼─────┐
    │ 🤖 Agent  │         │ 🎭 Persona   │        │ 🧠 Memory │
    │ ReAct     │────────▶│ 6 Profiles   │───────▶│ Phantom DB│
    │ Loop      │         │ Model Bind   │        │ Prefetch  │
    └─────┬─────┘         └──────────────┘        └─────┬─────┘
          │                                              │
    ┌─────▼─────┐         ┌──────────────┐        ┌─────▼─────┐
    │ 🔧 Skill  │         │ 🔌 PSC       │        │ ⚖️ Weight │
    │ Registry  │         │ 4+ Providers │        │ Engine    │
    │ MCP       │         │ Auto-Fallback│        │ Explain   │
    └───────────┘         └──────────────┘        └───────────┘
```

---

## 🎭 6 Personas

| | Persona | Model | Temp | Speciality |
|--|---------|-------|------|------------|
| 🏛️ | **boi** | claude-sonnet-5 | 0.4 | Architecture & Strategy |
| 🔄 | **kamkaew** | gpt-4.1-mini | 0.5 | Runtime Orchestrator _(default)_ |
| 🧠 | **kampun** | claude-sonnet-5 | 0.3 | Root Cause & Pattern Analysis |
| 🐛 | **dang** | gpt-4.1-mini | 0.2 | Debug & Code Analysis |
| 📝 | **don** | gpt-4.1-nano | 0.5 | Documentation Specialist |
| 🎨 | **kine** | gpt-4o | 0.8 | Creative & UI/UX Design |

```bash
boi persona list          # View all
boi persona switch dang   # Switch persona
boi ask "debug this" --persona kampun
```

---

## 🧠 Phantom DB — Living Memory

```
ทุก conversation → extract facts → save to Phantom DB
                                    │
                              ┌─────▼─────┐
                              │ ⚖️ Weight  │
                              │  Engine    │
                              └─────┬─────┘
                                    │
              ┌─────────────────────┼─────────────────────┐
              ▼                     ▼                     ▼
        Truth Weight          Confidence          Recency Decay
         (evidence)           (conflicts)         (time-based)
              │                     │                     │
              └─────────────────────┼─────────────────────┘
                                    ▼
                           Final Score 0.72
```

```bash
boi weight explain "login-fix"
# Truth       : some evidence
# Confidence  : no conflicts
# Importance  : importance score
# Recency     : seen recently
# Usage       : usage frequency
# ────────────────────
# Final       0.72
```

---

## 🎮 All Commands

| Command | Description |
|---------|-------------|
| `boi` | **Launch TUI** (no args) |
| `boi ask "..."` | Ask AI agent |
| `boi run "..."` | Execute shell command |
| `boi init` | Initialize workspace |
| `boi config` | Show configuration |
| `boi persona list` | List personas |
| `boi persona switch <name>` | Switch persona |
| `boi skill list` | List skills |
| `boi skill init` | Initialize skills |
| `boi memory search "..."` | Search Phantom DB |
| `boi memory stats` | Memory statistics |
| `boi memory repomap` | Scan project structure |
| `boi weight explain <id>` | Weight breakdown |

---

## 🧬 Chimera DNA

```
OpenCode  ──→ Agent System · Permission · Event Bus ──┐
Hermes    ──→ GEPA Evolution · Multi-Terminal ─────────┤
Claude    ──→ Memory · Extended Thinking ──────────────┤
Codex     ──→ Sandbox · Approval Policy ───────────────┤
Antigravity ─→ Multi-Provider · Auto-Router ───────────┤──→ BOI CLI
Agent 0   ──→ Computer-as-Tool · Adaptive Memory ──────┤
ZeroClaw  ──→ Trait-Based · Security-First ────────────┘
```

> ไม่ใช่ Fork — คือการสกัด **Pattern, Architecture, UX** จากทุก CLI
> มาประกอบเป็นสถาปัตยกรรมใหม่

---

## 🏛️ Greek God Levels

```
🌌 Aether     Lv9 — Transcendence
⚡ Zeus       Lv8 — System Governance
🏹 Artemis    Lv7 — Autonomous Operation
☀️ Apollo     Lv6 — Mastery & Creation
🦉 Athena     Lv5 — Strategic Wisdom
👟 Hermes     Lv4 — Speed & Connection
🔨 Hephaestus Lv3 — Crafting & Building
🔥 Prometheus Lv2 — Learning & Discovery
⏳ Cronus     Lv1 — Foundation & Awareness
```

> แต่ละระดับมี 3 Tiers: 🔸 Low → 🔹 Mid → 💎 High

---

## 🗺️ Development Timeline

```
Phase 1  ████████  Core Runtime         ✅
Phase 2  ████████  PSC Mini             ✅
Phase 3  ████████  Persona              ✅
Phase 4  ████████  Skill                ✅
Phase 5  ████████  Memory + Phantom DB  ✅
Phase 6.1 ███████  Weight Engine        ✅
Phase 6.2 ███████  Agent Core           ✅
TUI      ███████  Bubbletea Terminal    ✅
────────────────────────────────────────────
         BOI CLI v1.0 🚀
```

---

## 📁 Project Structure

```
BOI-CLI/
├── cmd/boi/              # Entry point + TUI
├── internal/
│   ├── agent/            # ReAct Loop, Planner, Executor, Reviewer
│   ├── cli/              # Cobra commands (ask, run, init, config, ...)
│   ├── command/          # Shell executor + sandbox
│   ├── config/           # YAML config loader
│   ├── llm/              # Provider Supply Chain (PSC)
│   │   ├── providers/    # OpenAI, Anthropic implementations
│   │   └── factory/      # Env-based provider factory
│   ├── mcp/              # MCP client
│   ├── memory/           # Phantom DB + Prefetch + Hook
│   ├── persona/          # 6 Persona profiles
│   ├── skill/            # Skill registry + loader
│   ├── tui/              # Bubbletea TUI components
│   ├── weight/           # Weight Engine + Explain Mode
│   └── workspace/        # Project detection + scanner
├── .boi/                 # User workspace (config, personas, skills, memory)
├── bin/                  # Compiled binary
├── .env.example          # PSC provider template
├── go.mod                # Go module
├── Makefile              # Build targets
└── PLAN.md               # Full architecture document
```

---

## 🔧 Tech Stack

| Layer | Technology | Why |
|-------|-----------|-----|
| **Language** | Go 1.26 | Single binary, goroutines, fast |
| **CLI** | Cobra + Viper | Standard Go CLI framework |
| **TUI** | Bubbletea + Lipgloss + Glamour | Elm architecture, beautiful |
| **LLM** | Direct HTTP | Multi-provider, no lock-in |
| **Memory** | JSON files | Zero external deps |
| **Config** | YAML | Human-readable |

---

## 🚀 Getting Started

```bash
# 1. Clone
git clone https://github.com/wersoul-source/BOI-CLI.git
cd BOI-CLI

# 2. Build
go build -o bin/boi.exe ./cmd/boi

# 3. Init
./bin/boi init

# 4. Launch TUI
./bin/boi

# 5. Or ask directly
./bin/boi ask "explain this project"
```

> 💡 **Tip:** Add `bin/` to your PATH to use `boi` from anywhere.

---

<p align="center">
  <b>BOI CLI</b> — Built with ❤️ by <b>Kampun (คำปัน)</b> & the <b>BOI Family</b><br>
  <sub>Approved by หัวหน้าเอ · สร้าง 27 กรกฎาคม 2026</sub>
</p>
