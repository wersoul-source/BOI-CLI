# 📖 BOI CLI — Full Command Reference

> เวอร์ชัน: v0.1.0
> อัปเดต: 31 กรกฎาคม 2026

---

## Overview

BOI CLI มี 7 top-level commands:

```
boi              Launch TUI
boi ask          Ask AI agent (ReAct loop with memory + persona)
boi run          Execute shell command
boi init         Initialize BOI workspace
boi config       View/edit configuration
boi persona      Manage BOI Family personas
boi skill        Manage skills
boi memory       Manage Phantom DB memory
boi weight       Weight Engine operations
```

---

## 1. `boi` — Launch TUI

```
boi [no arguments]
```

เปิด full-screen terminal interface (Bubbletea-based).

| Key | Action |
|-----|--------|
| `Tab` | Switch persona |
| `Enter` | Send message |
| `Ctrl+N` | New line |
| `Ctrl+Q` | Quit |

**Examples:**
```bash
boi                    # Launch TUI
boi --version          # Show version
boi --help             # Show help
```

---

## 2. `boi ask` — Ask AI Agent

```
boi ask [query] [flags]
```

Runs full agent loop: Plan → Execute → Review → Learn. Injects memory context + persona profile.

| Flag | Short | Default | Description |
|------|-------|---------|-------------|
| `--persona` | `-p` | `kamkaew` | Persona to use (boi, kamkaew, kampun, dang, don, kine) |
| `--steps` | `-s` | `15` | Max agent steps per query |
| `--verbose` | `-v` | `false` | Verbose output (show steps, tokens, time) |

**Examples:**
```bash
boi ask "hello"
boi ask "debug the login flow" -p dang -v
boi ask "explain this architecture" -p boi -s 20
boi ask "find the root cause" -p kampun --verbose
```

**Note:** Without `.env` configured (PSC_* variables), `boi ask` runs in simulated response mode. Real AI responses require at least one provider configured.

---

## 3. `boi run` — Execute Shell Command

```
boi run [command] [flags]
```

Executes shell commands with sandbox safety checks.

| Flag | Short | Default | Description |
|------|-------|---------|-------------|
| `--verbose` | `-v` | `false` | Verbose output |
| `--dir` | `-d` | (workspace root) | Working directory |

**Examples:**
```bash
boi run "git status"
boi run "ls -la" -v
boi run "npm test" -d ./frontend
boi run "go build ./cmd/boi"
```

---

## 4. `boi init` — Initialize Workspace

```
boi init [flags]
```

Creates `.boi/` directory with default configuration, persona profiles, and memory storage.

| Flag | Short | Default | Description |
|------|-------|---------|-------------|
| `--force` | `-f` | `false` | Overwrite existing config |

**What it creates:**
```
.boi/
├── config.yaml       # Main configuration
├── personas/         # 6 persona profiles
├── skills/           # Skill definitions
├── memory/           # Phantom DB (SQLite)
├── .gitignore        # .boi/ entry
└── memory.md         # Project memory file
```

**Examples:**
```bash
boi init                # Initialize workspace
boi init --force        # Overwrite existing
```

---

## 5. `boi config` — View/Edit Configuration

```
boi config [flags]
```

Displays current BOI CLI configuration.

| Flag | Short | Default | Description |
|------|-------|---------|-------------|
| `--all` | `-a` | `false` | Show full config including API keys (masked) |

**Examples:**
```bash
boi config              # Summary view
boi config --all        # Full YAML output
```

**Summary output:**
```
┌─────────────────────────────────────┐
│        BOI CLI Config               │
├─────────────────────────────────────┤
│ Provider:   openai                  │
│ Model:      gpt-4.1-mini            │
│ Log Level:  info                    │
│ Workspace:  /path/to/project        │
│ API Keys:   1 configured            │
└─────────────────────────────────────┘
```

---

## 6. `boi persona` — Manage Personas

### 6a. `boi persona list`

```
boi persona list
```

Lists all 6 personas with model, temperature, and description. Active persona marked with `*`.

**Output:**
```
NAME      MODEL            TEMP  DESCRIPTION
────      ─────            ────  ───────────
 boi      claude-sonnet-5   0.4  Architecture & Strategy
* kamkaew  gpt-4.1-mini     0.5  Runtime Orchestrator
 kampun   claude-sonnet-5   0.3  Root Cause & Pattern Analysis
 dang     gpt-4.1-mini      0.2  Debug & Code Specialist
 don      gpt-4.1-nano      0.5  Documentation Specialist
 kine     gpt-4o            0.8  UI/UX & Creative Design
```

### 6b. `boi persona switch <name>`

```
boi persona switch [name]
```

Switch active persona. Updates `.boi/config.yaml`.

**Examples:**
```bash
boi persona switch dang       # Switch to debug specialist
boi persona switch kampun     # Switch to root cause analyst
boi persona switch kamkaew    # Switch back to default
```

### 6c. `boi persona init`

```
boi persona init
```

Copy default persona YAML files to `.boi/personas/`.

---

## 7. `boi skill` — Manage Skills

### 7a. `boi skill list`

```
boi skill list
```

Lists all loaded skills from `.boi/skills/`.

**Examples:**
```bash
boi skill list
# Output:
# BOI Skills
# ----------
#   git          — Git operations assistant
#   web          — Web search and fetch
```

### 7b. `boi skill init`

```
boi skill init
```

Initialize default skills (`git.skill.md`, `web.skill.md`) in `.boi/skills/`.

### 7c. `boi skill show <name>`

```
boi skill show [name]
```

Show full content of a skill definition.

**Examples:**
```bash
boi skill show git           # Show git skill details
boi skill show web           # Show web skill details
```

---

## 8. `boi memory` — Manage Phantom DB

### 8a. `boi memory search <query>`

```
boi memory search [query]
```

Search Phantom DB for relevant memories. Uses FTS5 full-text search.

**Example:**
```bash
boi memory search "login"
# Found 3 memories:
# 1. [solution] login-fix (score: 0.72)
#    Fixed by URL-encoding password in auth.js line 45
# 2. [fact] login-bug (score: 0.65)
#    Login page crashes when password has special chars
```

### 8b. `boi memory stats`

```
boi memory stats
```

Show Phantom DB statistics (total entries, size, memory types).

### 8c. `boi memory save <type> <key> <content>`

```
boi memory save [type] [key] [content]
```

Manually save a memory entry.

**Example:**
```bash
boi memory save fact "port-config" "Production server uses port 3055"
boi memory save solution "db-timeout" "Increase connection timeout to 30s"
```

### 8d. `boi memory repomap`

```
boi memory repomap
```

Scan and display project structure summary (file tree + sizes).

### 8e. `boi memory init`

```
boi memory init
```

Initialize `.boi/memory.md` project memory file.

---

## 9. `boi weight` — Weight Engine

### 9a. `boi weight explain <id>`

```
boi weight explain [memory-id-pattern]
```

Show weight breakdown for a memory entry. Explains how the 10-dimension score is calculated.

**Example:**
```bash
boi weight explain "mem_1785403325978479400"
# ┌──────────────────────────────┐
# │ WEIGHT EXPLANATION           │
# ├──────────────────────────────┤
# │ Entry: login-fix             │
# │ Type:  solution              │
# │                              │
# │ Access:    0.85  (high)      │
# │ Recency:   0.60  (3 days)    │
# │ Relevance: 0.70  (matched)   │
# └──────────────────────────────┘
```

---

## Command Tree (Full)

```
boi
├── ask <query> [-p persona] [-s steps] [-v]
├── run <command> [-v] [-d dir]
├── init [-f]
├── config [-a]
├── persona
│   ├── list
│   ├── switch <name>
│   └── init
├── skill
│   ├── list
│   ├── init
│   └── show <name>
├── memory
│   ├── search <query>
│   ├── stats
│   ├── save <type> <key> <content>
│   ├── repomap
│   └── init
└── weight
    └── explain <id>
```

---

## Exit Codes

| Code | Meaning |
|------|---------|
| `0` | Success |
| `1` | General error |
| `2` | Invalid arguments / flag |
| `3` | Config not found (run `boi init` first) |
| `4` | Provider error (check .env) |

---

## Environment Variables

| Variable | Description |
|----------|-------------|
| `BOI_HOME` | Override `.boi/` directory location |
| `BOI_LOG_LEVEL` | Override log level (debug/info/warn/error) |
| `PSC_1_NAME` | Provider 1 name (openai/anthropic/deepseek/ollama) |
| `PSC_1_API_KEY` | Provider 1 API key |
| `PSC_1_BASE_URL` | Provider 1 base URL (optional, for proxies/custom endpoints) |
| `PSC_1_MODEL` | Provider 1 model (gpt-4.1-mini/claude-sonnet-5/etc) |
| `PSC_2_*` through `PSC_4_*` | Fallback providers 2–4 |

---

## Future Commands (Roadmap)

| Command | Description | Target |
|---------|-------------|--------|
| `boi doctor` | Health check (Go ver, binary, config, PSC, memory) | v0.2.0 |
| `boi upgrade` | Self-update to latest version | v0.2.0 |
| `boi evolve` | Show evolution score and progress | v0.3.0 |
| `boi doctor --fix` | Auto-fix common issues | v0.3.0 |
| `boi plugin install` | Install third-party plugins | v0.4.0 |
| `boi agent orchestrate` | Multi-agent orchestration | v0.5.0 |

---

*สิ้นสุด CLI_COMMANDS.md*
