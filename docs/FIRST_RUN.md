# 🚀 BOI CLI — First Run Experience

> ออกแบบ: 31 กรกฎาคม 2026
> โดย: Kampun (คำปัน) — BOI Family

---

## Philosophy: Zero Friction, Maximum Delight

BOI CLI's first run ต้อง "just works" — ไม่ต้อง config, ไม่ต้อง login, ไม่ต้อง setup อะไรเลย เพื่อให้ user ได้เห็นคุณค่าภายใน 30 วินาทีแรก

```
Install → boi → Splash → TUI → Working!
                   (< 1 second)
```

---

## First Run Flow

```
User runs: boi (for the first time)
         │
         ▼
┌─────────────────────────────────────────────────────────────────┐
│  STEP 0: BINARY CHECK                                           │
│                                                                 │
│  Check:   Is $HOME/.boi/ initialized?                           │
│  Result:  No → trigger first-run sequence                       │
│           Yes → normal launch                                   │
└──────────────────────┬──────────────────────────────────────────┘
                       ▼
┌─────────────────────────────────────────────────────────────────┐
│  STEP 1: CHECK GO INSTALLATION                                  │
│                                                                 │
│  Run:   go version                                              │
│  If OK: "Go 1.24.2 detected" ✅                                 │
│  If no: "Go not found. BOI CLI works standalone,               │
│          but 'go install' method requires Go."                   │
│         (ไม่ error — แค่แจ้งเตือนแบบ soft warning)                │
└──────────────────────┬──────────────────────────────────────────┘
                       ▼
┌─────────────────────────────────────────────────────────────────┐
│  STEP 2: CREATE .boi/ WORKSPACE                                 │
│                                                                 │
│  Creates silently (no prompts, defaults):                       │
│                                                                 │
│  .boi/                                                          │
│  ├── config.yaml         (default provider: simulated)          │
│  ├── personas/                                                 │
│  │   ├── boi.yaml        (claude-sonnet-5, temp 0.4)           │
│  │   ├── kamkaew.yaml    (gpt-4.1-mini, temp 0.5) ★ active     │
│  │   ├── kampun.yaml     (claude-sonnet-5, temp 0.3)           │
│  │   ├── dang.yaml       (gpt-4.1-mini, temp 0.2)              │
│  │   ├── don.yaml        (gpt-4.1-nano, temp 0.5)              │
│  │   └── kine.yaml       (gpt-4o, temp 0.8)                    │
│  └── skills/                                                   │
│      ├── git.skill.md                                          │
│      └── web.skill.md                                           │
│                                                                 │
│  Message: "✅ .boi/ workspace created"                          │
└──────────────────────┬──────────────────────────────────────────┘
                       ▼
┌─────────────────────────────────────────────────────────────────┐
│  STEP 3: SPLASH SCREEN                                          │
│                                                                 │
│  ┌─────────────────────────────────────────────────┐            │
│  │                                                 │            │
│  │    888888b.    .d88888b.  8888888               │            │
│  │    888  "88b  d88P" "Y88b   888                 │            │
│  │    888  .88P  888     888   888                 │            │
│  │    8888888K.  888     888   888                 │            │
│  │    888  "88b  888     888   888                 │            │
│  │    888  .88P  888     888   888                 │            │
│  │    8888888K.  Y88b. .d88P   888                 │            │
│  │    888   Y88b  "Y88888P"  8888888               │            │
│  │                                                 │            │
│  │       BOI CLI v0.1.0                            │            │
│  │       AI Agent Runtime — BOI Family              │            │
│  │                                                 │            │
│  │  ┌───────────────────────────────────────────┐  │            │
│  │  │  Status: Ready                            │  │            │
│  │  │  Persona: kamkaew (Orchestrator)           │  │            │
│  │  │  Provider: Simulated (no API keys)        │  │            │
│  │  │  Memory: 0 entries                        │  │            │
│  │  └───────────────────────────────────────────┘  │            │
│  │                                                 │            │
│  │  Quick Setup:                                   │            │
│  │    cp .env.example .env                         │            │
│  │    # Add PSC_1_API_KEY=sk-...                   │            │
│  │                                                 │            │
│  │  Press Enter to continue...                     │            │
│  └─────────────────────────────────────────────────┘            │
└──────────────────────┬──────────────────────────────────────────┘
                       ▼
┌─────────────────────────────────────────────────────────────────┐
│  STEP 4: .env PROMPT (if no providers)                          │
│                                                                 │
│  BOI detects: no PSC_* variables found                          │
│                                                                 │
│  "No LLM providers configured.                                  │
│   BOI will run in simulated mode until you set up a provider.   │
│                                                                 │
│   To set up now, choose a provider:"                             │
│                                                                 │
│   [1] OpenAI (gpt-4.1-mini)                                     │
│   [2] Anthropic (claude-sonnet)                                 │
│   [3] Local (Ollama)                                            │
│   [4] Skip — use simulated mode                                 │
│                                                                 │
│   Your choice: _                                                │
│                                                                 │
│  If user picks 1-3:                                             │
│    "Enter API key:"  → saved to .env                            │
│    "Enter model: [default]" → saved                             │
│                                                                 │
│  If user picks 4:                                               │
│    "OK — BOI will run in simulated mode.                        │
│     You can set up providers later in .env"                     │
└──────────────────────┬──────────────────────────────────────────┘
                       ▼
┌─────────────────────────────────────────────────────────────────┐
│  STEP 5: RUN DOCTOR                                             │
│                                                                 │
│  Health check results:                                          │
│                                                                 │
│  ┌───────────────────────────────────────────────┐              │
│  │  🩺 BOI Doctor — Health Check                  │              │
│  ├───────────────────────────────────────────────┤              │
│  │  ✅ Binary:     ~/.boi/bin/boi (v0.1.0)      │              │
│  │  ✅ Workspace:  .boi/ initialized             │              │
│  │  ✅ Config:     .boi/config.yaml valid        │              │
│  │  ✅ Personas:   6 loaded                      │              │
│  │  ✅ Skills:     2 loaded (git, web)           │              │
│  │  ✅ Memory:     Phantom DB ready              │              │
│  │  ⚠️  Provider:   No API keys (simulated mode) │              │
│  │                                               │              │
│  │  Overall: READY ⚠️ (no real AI yet)           │              │
│  └───────────────────────────────────────────────┘              │
│                                                                 │
│  (If all ✅ including provider: "Overall: HEALTHY ✅")           │
└──────────────────────┬──────────────────────────────────────────┘
                       ▼
┌─────────────────────────────────────────────────────────────────┐
│  STEP 6: READY TO USE                                           │
│                                                                 │
│  ┌──────────────────────────────────────────────────┐           │
│  │                                                  │           │
│  │  ✅ BOI CLI is ready!                            │           │
│  │                                                  │           │
│  │  Try these commands:                             │           │
│  │                                                  │           │
│  │    boi                  → Launch TUI             │           │
│  │    boi ask "hello"     → Chat with AI           │           │
│  │    boi ask -p dang "debug this"                  │           │
│  │    boi memory search "setup"                     │           │
│  │    boi --help           → All commands           │           │
│  │                                                  │           │
│  │  Need real AI?                                   │           │
│  │    cp .env.example .env && edit .env            │           │
│  │                                                  │           │
│  └──────────────────────────────────────────────────┘           │
│                                                                 │
│  User presses Enter → TUI launches                              │
└─────────────────────────────────────────────────────────────────┘
```

---

## Simplified Flow: Pure CLI Users

If user runs a CLI command first (not TUI):

```bash
boi ask "hello"  # First run
```

```
→ .boi/ not found → auto-init silently
→ No providers → simulated mode
→ Response: "[BOI · kamkaew · simulated] Hello! ..."
→ Bottom line: "Tip: Set PSC_1_API_KEY in .env for real AI responses"
```

No prompts, no interruptions. Just works + gentle tip.

---

## What Users See By Method

### Method A: `boi` (TUI) — First Time
```
→ Auto-init workspace
→ Splash screen (3s auto-close or Enter)
→ .env prompt (if no provider)
→ Doctor result
→ TUI chat interface
```

### Method B: `boi ask "hello"` — First Time
```
→ Auto-init workspace (silent)
→ Response (simulated if no .env)
→ Gentle tip at bottom
```

### Method C: `boi init` — Explicit Initialize
```
→ Creates .boi/ workspace
→ Shows what was created
→ Gives next steps
```

### Method D: `go install` User
```
$ go install github.com/boi-family/boi-cli/cmd/boi@latest
$ boi
→ Same first-run experience as Method A
```

---

## Error Recovery During First Run

| Scenario | Behavior |
|----------|----------|
| No internet (can't check latest version) | Skip version check, use installed binary |
| Can't create .boi/ (permissions) | Show error + suggest `sudo` or manual path |
| .boi/ already exists (from another tool) | Detect + ask before overwriting |
| No .env.example found | Skip provider setup prompt |
| Terminal doesn't support TUI | Fallback to CLI-only mode |

---

## Success Metrics (First Run)

| Metric | Target |
|--------|--------|
| Time from `boi` to splash screen | < 500ms |
| Time from `boi` to first response | < 1s |
| Steps required before first use | 0 (zero config) |
| Steps required for real AI | 2 (cp .env.example + add key) |
| % users who complete first run | > 95% |
| % users who set up provider on first run | > 60% |

---

## Anti-Patterns We Avoid

| Anti-Pattern | Why Avoiding |
|-------------|--------------|
| **"Login required" gate** | claude/gh block you until login → friction |
| **"Run setup wizard"** | Interrupts flow with 10 questions |
| **"Install dependencies first"** | Go, Node, Python — no prerequisites |
| **"Sign up for account"** | No account system, just API keys |
| **"Check your email to verify"** | No email verification flow |
| **Error on first run without config** | Always fallback gracefully |

---

*สิ้นสุด FIRST_RUN.md*
