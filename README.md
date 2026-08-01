<p align="center">
  <img src="https://img.shields.io/badge/BOI_CLI-0.1.2-6C63FF?style=for-the-badge" alt="BOI CLI">
  <br>
  <img src="https://img.shields.io/github/v/release/wersoul-source/BOI-CLI?color=6C63FF" alt="Release">
  <img src="https://img.shields.io/badge/Go-1.24-00ADD8?style=flat&logo=go" alt="Go">
  <img src="https://img.shields.io/badge/Platform-Win%20|%20Mac%20|%20Linux-lightgrey?style=flat" alt="Platform">
  <img src="https://img.shields.io/badge/License-MIT-blue?style=flat" alt="License">
</p>

<p align="center">
  <img src="assets/boi-logo.svg" alt="BOI CLI" width="550">
</p>

<h3 align="center">An AI team of 6 personas in your terminal. One binary. No server.</h3>

---

## 🚀 Get Started

### 1. Install
```bash
# Download latest release (Windows)
Invoke-WebRequest -Uri "https://github.com/wersoul-source/BOI-CLI/releases/download/v0.1.2/boi_0.1.2_windows_amd64.tar.gz" -OutFile boi.tar.gz
tar -xzf boi.tar.gz && cd boi_0.1.2_windows_amd64
.\boi.exe init
```

### 2. Setup Providers
```bash
boi setup
```
Pick from 10 providers (OpenAI, Anthropic, Google, Groq, DeepSeek, Mistral, xAI, Ollama, OpenRouter, Together) or add custom. Add 2+ for auto-fallback.

### 3. Choose Persona
```bash
boi persona wizard
```
Each persona introduces itself — pick your thinking style.

### 4. Launch!
```bash
boi                        # Splash → Press Enter → Chat (TUI)
boi ask "hello" --verbose  # Or CLI mode
```

> No API keys? BOI runs in simulated mode.

---

## ✨ Features

| | |
|---|---|
| 🧠 **Agent Loop** | ReAct pattern — plan → execute → review → learn, cross-session memory |
| 🎭 **6 Personas** | Switch between specialized AI profiles for architecture, debug, docs, creative |
| 🔌 **PSC Fallback** | Provider Supply Chain — chain up to 4 LLM providers with automatic failover |
| 💾 **Phantom DB** | File-based memory with weight engine — remembers what matters across sessions |
| 🖥️ **TUI + CLI** | Full-screen Bubbletea terminal UI or direct CLI commands — your choice |

---

## 🧩 Personas

| Persona | Role | Model | Temp |
|---------|------|-------|------|
| **kamkaew** | Runtime Orchestration *(default)* | gpt-4.1-mini | 0.5 |
| **boi** | Architecture & System Design | claude-sonnet-5 | 0.4 |
| **kampun** | Knowledge Mining & Pattern Analysis | claude-sonnet-5 | 0.3 |
| **dang** | Debug & Code Specialist | gpt-4.1-mini | 0.2 |
| **don** | Research & Documentation | gpt-4.1-nano | 0.5 |
| **kine** | Creative Design & Imagination | gpt-4o | 0.8 |

```bash
boi persona list              # View all
boi persona switch dang       # Switch persona
boi ask "debug this" -p dang  # Use for one query
```

---

## 📖 Commands

| Command | Description |
|---------|-------------|
| `boi` | Launch TUI (full-screen terminal) |
| `boi ask "..."` | AI agent query (ReAct loop) |
| `boi run "..."` | Execute shell command |
| `boi init` | Initialize workspace |
| `boi doctor` | System health check |
| `boi persona list/switch` | Manage personas |
| `boi skill list/init` | Manage skills |
| `boi memory search/stats` | Phantom DB memory |
| `boi weight explain <id>` | Memory weight breakdown |
| `boi upgrade` | Self-update to latest |
| `boi version` | Show version |

---

## 📸 Screenshot

```
┌─ BOI CLI — 🔨 Hephaestus | Mid ── Persona: kampun ─── idle ────────┐
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
│  └──────────────────────────────────────────────────────────────┘  │
│                                                                     │
│  ◆ BOI: Root cause — special characters in password                │
│                                                                     │
│  ✅ 2 steps  ·  450 tokens  ·  0.3s                                 │
├─────────────────────────────────────────────────────────────────────┤
│ > login bug มันเกิดจากอะไรครับ                                      │
├─────────────────────────────────────────────────────────────────────┤
│  Tab:persona  Enter:send  Ctrl+Q:quit                               │
└─────────────────────────────────────────────────────────────────────┘
```

---

## ❓ FAQ

**Do I need a server or database?** No. Single Go binary. Everything lives in `.boi/`.

**Does it work without an API key?** Yes — simulated mode. Add `PSC_*` in `.env` for real AI.

**How is this different from Claude Code / Codex CLI?** 6 specialized personas, cross-session memory with weight engine, 4-provider auto-fallback — all as a single binary.

---

<p align="center">
  <a href="https://github.com/wersoul-source/BOI-CLI">
    <img src="https://img.shields.io/github/stars/wersoul-source/BOI-CLI?style=social" alt="Stars">
  </a>
</p>

<p align="center">
  <b>BOI CLI</b> — Built by <b>Kampun (คำปัน)</b> &amp; the <b>BOI Family</b><br>
  <sub>MIT License · July 2026</sub>
</p>
