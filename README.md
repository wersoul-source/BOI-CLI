

<p align="center">
  <img src="https://img.shields.io/badge/BOI_CLI-0.1.0-6C63FF?style=for-the-badge" alt="BOI CLI">
  <br>
  <img src="https://img.shields.io/badge/Go-1.24-00ADD8?style=flat&logo=go" alt="Go">
  <img src="https://img.shields.io/badge/Binary-~8MB-success?style=flat" alt="Binary">
  <img src="https://img.shields.io/badge/Platform-Win%20%7C%20Mac%20%7C%20Linux-lightgrey?style=flat" alt="Platform">
  <img src="https://img.shields.io/badge/License-MIT-blue?style=flat" alt="License">
</p>

<h3 align="center">One command. One workspace. One AI runtime.</h3>

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

BOI CLI is the **BOI Family's AI Agent Runtime** — a single Go binary that runs in your terminal. No server. No Docker. No database. Just you and the agent.

---

## ✨ Features

| | |
|---|---|
| 🧠 **Agent Loop** | ReAct pattern — plan → execute → review → learn |
| 🎭 **6 Personas** | Switch between specialized AI profiles (boi, kamkaew, kampun, dang, don, kine) |
| 💾 **Phantom DB** | Cross-session memory with weight engine — remembers what matters |
| 🔌 **PSC** | Provider Supply Chain — 4 LLM providers with auto-fallback |
| 🖥️ **TUI + CLI** | Full-screen Bubbletea terminal UI, or direct CLI commands |

---

## ⚡ Quick Start

```bash
git clone https://github.com/wersoul-source/BOI-CLI.git
cd BOI-CLI
go build -o bin/boi ./cmd/boi
./bin/boi init
```

---

## 📸 TUI

```
┌─ BOI CLI — 🔨 Hephaestus | Mid ── Persona: kampun ─── idle ────────┐
│                   █████████████████████████████████████              │
├─────────────────────────────────────────────────────────────────────┤
│                                                                     │
│  ▶ You: login bug เกิดจากอะไร                                       │
│                                                                     │
│  ◆ BOI: 🔍 Phantom DB พบ 3 ความจำที่เกี่ยวข้อง...                   │
│                                                                     │
│  ┌─ <memory-context> ───────────────────────────────────────────┐  │
│  │ [solution] login-fix   weight: 0.72                          │  │
│  │   Fixed by URL-encoding password in auth.js line 45          │  │
│  │ [fact]     login-bug   weight: 0.65                          │  │
│  │   Login page crashes when password has special characters    │  │
│  └──────────────────────────────────────────────────────────────┘  │
│                                                                     │
│  ◆ BOI: สาเหตุคือ special characters ใน password                    │
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

## 🧩 Personas

| | Persona | Role | Model | Temp |
|---|---------|------|-------|------|
| 🏛️ | **boi** | Architecture & Strategy | claude-sonnet-5 | 0.4 |
| 🔄 | **kamkaew** | Runtime Orchestrator *(default)* | gpt-4.1-mini | 0.5 |
| 🧠 | **kampun** | Root Cause & Pattern Analysis | claude-sonnet-5 | 0.3 |
| 🐛 | **dang** | Debug & Code Specialist | gpt-4.1-mini | 0.2 |
| 📝 | **don** | Documentation Specialist | gpt-4.1-nano | 0.5 |
| 🎨 | **kine** | UI/UX & Creative Design | gpt-4o | 0.8 |

```bash
boi persona list            # View all personas
boi persona switch dang     # Switch active persona
boi ask "debug this" -p kampun  # Use specific persona
```

---

## 🔌 Providers

BOI uses **PSC** (Provider Supply Chain) — chain up to 4 LLM providers with automatic fallback.

```bash
cp .env.example .env
```

Edit `.env` and add at least one provider:

```env
PSC_1_NAME=openai
PSC_1_API_KEY=sk-...
PSC_1_MODEL=gpt-4.1-mini
```

Supports: **OpenAI**, **Anthropic**, and any OpenAI-compatible endpoint.

> ⚠️ Without `.env` configured, `boi ask` runs in simulated response mode.

---

## 📖 Commands

| Command | Description |
|---------|-------------|
| `boi` | Launch TUI (no args) |
| `boi ask "..."` | Ask AI agent |
| `boi run "..."` | Execute shell command |
| `boi init` | Initialize workspace |
| `boi config` | Show configuration |
| `boi persona list` | List all personas |
| `boi persona switch <name>` | Switch active persona |
| `boi skill list` | List loaded skills |
| `boi memory search "..."` | Search Phantom DB |
| `boi weight explain <id>` | Weight breakdown |

---

## 🗺️ Roadmap

```
Phase 1  ████████████  Core Runtime         ✅
Phase 2  ████████████  PSC (Multi-LLM)      ✅
Phase 3  ████████████  Persona System       ✅
Phase 4  ████████████  Skill Registry       ✅
Phase 5  ████████████  Phantom DB Memory    ✅
Phase 6  ████████████  Agent Loop + Weight  ✅
TUI      ████████████  Bubbletea Terminal   ✅
────────────────────────────────────────────────
         v0.1.0 — All core systems complete ✅
```

---

<p align="center">
  <b>BOI CLI</b> — Built with ❤️ by <b>Kampun (คำปัน)</b> & the <b>BOI Family</b><br>
  <sub>July 2026</sub>
</p>
