<p align="center">
  <img src="https://img.shields.io/badge/BOI_CLI-0.1.0-6C63FF?style=for-the-badge" alt="BOI CLI">
  <br>
  <img src="https://img.shields.io/badge/Go-1.24-00ADD8?style=flat&logo=go" alt="Go">
  <img src="https://img.shields.io/badge/Binary-~7MB-success?style=flat" alt="Binary">
  <img src="https://img.shields.io/badge/Platform-Win%20|%20Mac%20|%20Linux-lightgrey?style=flat" alt="Platform">
  <img src="https://img.shields.io/badge/License-MIT-blue?style=flat" alt="License">
</p>

<h3 align="center">An AI team of 6 personas in your terminal. One binary. No server.</h3>

```bash
git clone https://github.com/wersoul-source/BOI-CLI.git && cd BOI-CLI && go build -o bin/boi ./cmd/boi && ./bin/boi init
```

```
 888888b.    .d88888b.  8888888
 888  "88b  d88P" "Y88b   888
 888  .88P  888     888   888
 8888888K.  888     888   888
 888  "88b  888     888   888
 888  .88P  888     888   888
 8888888K.  Y88b. .d88P   888
 888   Y88b  "Y88888P"  8888888
```

---

## Quick Start

```bash
cp .env.example .env          # Add your API key
./bin/boi ask "hello"         # Test AI response
./bin/boi                     # Launch TUI
```

> No API keys? BOI runs in simulated mode — test the interface instantly.

---

## Features

| | |
|---|---|
| 🧠 **Agent Loop** | ReAct pattern — plan → execute → review → learn, cross-session memory |
| 🎭 **6 Personas** | Switch between specialized AI profiles for architecture, debug, docs, creative |
| 🔌 **PSC Fallback** | Provider Supply Chain — chain up to 4 LLM providers with automatic failover |
| 💾 **Phantom DB** | File-based memory with weight engine — remembers what matters across sessions |
| 🖥️ **TUI + CLI** | Full-screen Bubbletea terminal UI or direct CLI commands — your choice |

---

## Personas

| Persona | Role | Model | Temp |
|---------|------|-------|------|
| **kamkaew** | Orchestration *(default)* | gpt-4.1-mini | 0.5 |
| **boi** | Architecture & Strategy | claude-sonnet-5 | 0.4 |
| **kampun** | Root Cause & Pattern Analysis | claude-sonnet-5 | 0.3 |
| **dang** | Debug & Code Specialist | gpt-4.1-mini | 0.2 |
| **don** | Documentation Specialist | gpt-4.1-nano | 0.5 |
| **kine** | UI/UX & Creative Design | gpt-4o | 0.8 |

```bash
boi persona list              # View all
boi persona switch dang       # Switch persona
boi ask "debug this" -p dang  # Use for one query
```

---

## Commands

| Command | Description |
|---------|-------------|
| `boi` | Launch TUI (full-screen terminal) |
| `boi ask "..."` | AI agent query (ReAct loop) |
| `boi run "..."` | Execute shell command |
| `boi init` | Initialize workspace |
| `boi config` | View configuration |
| `boi persona list` | List all personas |
| `boi persona switch <name>` | Switch active persona |
| `boi skill list` | List loaded skills |
| `boi memory search "..."` | Search Phantom DB |
| `boi memory stats` | Memory statistics |
| `boi memory save <type> <key> <val>` | Manual memory entry |
| `boi memory repomap` | Repository structure scan |
| `boi weight explain <id>` | Memory weight breakdown |

---

## Screenshot

```
┌─ BOI CLI — 🔨 Hephaestus | Mid ── Persona: kampun ─── idle ────────┐
│                   █████████████████████████████████████              │
├─────────────────────────────────────────────────────────────────────┤
│                                                                     │
│  ▶ You: login bug เกิดจากอะไร                                       │
│                                                                     │
│  ◆ BOI: 🔍 Phantom DB found 3 related memories...                  │
│                                                                     │
│  ┌─ <memory-context> ───────────────────────────────────────────┐  │
│  │ [solution] login-fix   weight: 0.72                          │  │
│  │   Fixed by URL-encoding password in auth.js line 45          │  │
│  │ [fact]     login-bug   weight: 0.65                          │  │
│  │   Login page crashes when password has special characters    │  │
│  └──────────────────────────────────────────────────────────────┘  │
│                                                                     │
│  ◆ BOI: Root cause — special characters in password                │
│                                                                     │
│  ✅ 2 steps  ·  450 tokens  ·  0.3s                                 │
│                                                                     │
├─────────────────────────────────────────────────────────────────────┤
│ > login bug มันเกิดจากอะไรครับ                                      │
├─────────────────────────────────────────────────────────────────────┤
│  Tab:persona  Enter:send  Ctrl+N:newline  Ctrl+Q:quit               │
└─────────────────────────────────────────────────────────────────────┘
```

---

## FAQ

### Do I need a server or database?
No. BOI CLI is a single Go binary. Everything — memory, personas, skills — lives on your filesystem in `.boi/`.

### Does it work without an API key?
Yes. Without `.env` configured, `boi ask` runs in simulated response mode. Add at least one `PSC_*` provider to get real AI responses.

### How is this different from Claude Code or Codex CLI?
BOI CLI gives you **6 specialized personas** (not one), cross-session memory with a weight engine, and 4-provider fallback — all as a single binary. No npm, no Python, no Node.

### What LLM providers are supported?
OpenAI, Anthropic, and any OpenAI-compatible endpoint. Chain up to 4 providers — if one fails, BOI auto-falls-back to the next.

---

## Community

<p align="center">
  <a href="https://github.com/wersoul-source/BOI-CLI">
    <img src="https://img.shields.io/github/stars/wersoul-source/BOI-CLI?style=social" alt="GitHub Stars">
  </a>
</p>

<p align="center">
  <b>BOI CLI</b> — Built by <b>Kampun (คำปัน)</b> &amp; the <b>BOI Family</b><br>
  <sub>Licensed under MIT. &copy; July 2026</sub>
</p>
