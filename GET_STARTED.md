# Get Started with BOI CLI

## What is BOI CLI?

BOI CLI is an **AI Agent Runtime** that runs entirely in your terminal. It gives you a personal team of 6 AI personas — each specialized for different tasks — that can chat, debug code, execute commands, and remember everything you teach them across sessions.

No browser. No Electron. No subscription. Just a single binary.

---

## Do I need a server?

**No.** BOI CLI is a single Go binary. No server, no Docker, no database setup. Everything runs locally on your machine.

```
git clone → go build → ./bin/boi init → ./bin/boi
```

That's it.

---

## What do I need?

1. **Go 1.24+** — to build the binary
2. **API key** from any LLM provider (OpenAI, Anthropic, or compatible)

> The CLI works without API keys too — `boi ask` will fall back to simulated response mode so you can test the interface. Real AI responses require `.env` configured with `PSC_*` variables.

---

## How to install?

### 1. Clone the repo

```bash
git clone https://github.com/wersoul-source/BOI-CLI.git
cd BOI-CLI
```

### 2. Build

```bash
go build -o bin/boi ./cmd/boi
```

This creates a single binary at `bin/boi` (or `bin/boi.exe` on Windows). Takes about 30 seconds.

### 3. Initialize

```bash
./bin/boi init
```

Creates a `.boi/` folder in your current directory with config, personas, skills, and memory storage.

---

## How to configure?

### 4. Set up your LLM provider

```bash
cp .env.example .env
```

Edit `.env` and add at least one provider:

```env
PSC_1_NAME=openai
PSC_1_API_KEY=sk-your-api-key-here
PSC_1_MODEL=gpt-4.1-mini
```

You can chain up to 4 providers — if one fails, BOI automatically falls back to the next.

Supports: **OpenAI**, **Anthropic**, and any OpenAI-compatible API endpoint.

---

## First command?

### 5. Test it

```bash
./bin/boi ask "hello"
```

If `.env` is configured, you get a real AI response. If not, BOI runs in simulated mode so you can still verify everything works.

---

## TUI mode?

### 6. Launch the full terminal UI

```bash
./bin/boi
```

Just type `boi` with no arguments — you'll see a full-screen terminal interface with:
- Chat history with memory context
- Persona switching with `Tab` key
- Command input at the bottom
- System status bar at the top

| Key | Action |
|-----|--------|
| `Tab` | Switch persona |
| `Enter` | Send message |
| `Ctrl+N` | New line |
| `Ctrl+Q` | Quit |

---

## How to switch persona?

BOI has 6 specialized personas:

| Persona | Does |
|---------|------|
| **kamkaew** | Orchestration, task coordination *(default)* |
| **boi** | Architecture, system design |
| **kampun** | Root cause analysis, patterns |
| **dang** | Debugging, code tracing |
| **don** | Documentation, knowledge synthesis |
| **kine** | Creative design, UI/UX |

### In TUI

Press `Tab` to cycle through personas. The current persona is shown in the status bar.

### In CLI

```bash
boi ask "explain this architecture" -p boi
boi ask "find the bug in this code" -p dang
boi persona switch dang
```

---

## What commands are available?

```bash
boi                           # Launch TUI
boi ask "your question"       # Ask AI agent
boi run "git status"          # Execute shell command
boi init                      # Initialize workspace
boi config                    # View configuration
boi persona list              # List personas
boi persona switch <name>     # Switch active persona
boi skill list                # List loaded skills
boi memory search "query"     # Search Phantom DB
boi weight explain <id>       # Memory weight breakdown
```

---

## What actually works right now? (v0.1.0)

| Feature | Status |
|---------|--------|
| `boi` — TUI splash screen → chat interface | ✅ |
| `boi ask "..."` — CLI agent (simulated without API keys, real with PSC) | ✅ |
| `boi persona list / switch` — manage personas | ✅ |
| `boi memory search / save` — Phantom DB | ✅ |
| `boi run "..."` — shell command execution | ✅ |
| AI responses with `.env` PSC variables configured | ✅ |
| AI responses without API keys (simulated mode) | ⚠️ Test mode only |

---

## Need help?

- `/help` in TUI — show keyboard shortcuts
- `boi --help` — list all commands
- `boi config --all` — view full config
- Issues: [github.com/wersoul-source/BOI-CLI](https://github.com/wersoul-source/BOI-CLI)
