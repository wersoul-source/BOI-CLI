# Get Started with BOI CLI

Version 2.4. See the [README](README.md) for the full guide.

You need three things: the `boi` binary, a Provider (a hosted API or a model on
your own machine), and a short check that the Provider can drive the Agent.

## 1. Install

From source (works on Linux, macOS and Windows; needs Go 1.24 or later):

```powershell
git clone https://github.com/wersoul-source/BOI-CLI.git
cd BOI-CLI
go build -o boi.exe ./cmd/boi        # Linux/macOS: go build -o boi ./cmd/boi
.\boi.exe version
```

Release archives and the install scripts fetch the latest **release**, which can
be older than the source. Check `boi version`.

## 2. Connect a Provider

### A hosted Provider

```powershell
.\boi.exe setup      # interactive wizard; writes .env in the project root
```

### A model on your machine (Ollama, llama.cpp, LM Studio)

A server on `localhost` needs no API key. Use the Provider name `ollama` for any
local OpenAI-compatible server and point `PSC_1_BASE_URL` at it. In PowerShell:

```powershell
$env:PSC_1_NAME     = "ollama"
$env:PSC_1_BASE_URL = "http://localhost:11434/v1"   # llama.cpp: http://localhost:8080/v1   LM Studio: http://localhost:1234/v1
$env:PSC_1_MODEL    = "qwen2.5-coder:7b"            # the exact name your server lists
```

The same four `PSC_1_*` lines work in a `.env` file in the project folder.
`.env` is never readable by the Agent.

Give the model the context you want when you start the server, not in BOI:
llama.cpp `-c 65536`; Ollama `OLLAMA_CONTEXT_LENGTH=65536` before `ollama serve`
(or `PARAMETER num_ctx 65536` in a Modelfile); LM Studio, the context-length
setting. A 7B model with a 64k context does not fit entirely in 6 GB of video
memory, so expect part of it on the CPU and slow answers.

### Slow models need longer limits

The defaults suit a hosted API: 2 minutes per task and per request, 64,000 tokens
per task. For a slow local model:

```powershell
$env:BOI_TASK_TIMEOUT    = "15m"       # a whole task (10s to 2h)
$env:BOI_REQUEST_TIMEOUT = "10m"       # one request to the model (10s to 2h)
$env:BOI_MAX_TOKENS      = "200000"     # input plus output over the whole task
# also: BOI_MAX_TOOL_CALLS (default 8), BOI_MAX_STEPS (default 12)
```

A value that is not valid stops start-up and names the variable.

## 3. Qualify the Provider

BOI turns on Tools and Skills only for what the model proved it can do:

```powershell
.\boi.exe provider qualify ollama      # the Provider name from PSC_1_NAME
```

Each probe gets 30 seconds, or 3 minutes for a local server (the first one loads
the model). Read the list:

- `completion`, `reasoning`, `context` passed: it can chat.
- `tool_calling`, `tool_observation`, `authority` passed: the Agent may use Tools.
- `native_tool_schema` failed: fine. BOI falls back to text envelopes.
- `skill_calling` passed: Skills are offered.

A capability that failed or was `inconclusive` stays off. Run `qualify` again
after you change the model, the endpoint, or after 30 days.

## 4. Choose what the Agent can do

```powershell
.\boi.exe env use coding        # read, edit, write, run commands, all with approval
.\boi.exe env                   # what is composed and what is actually active
```

`minimal` is read-only. `boi env` shows each Block as `active`, `loaded` or
`declared`, with the reason when it is not active. `.boi/environment.json` is
the file behind it; Skills and MCP servers are added there too (README).

## 5. Use it

```powershell
.\boi.exe                       # the TUI: asks, edits and runs with your approval
.\boi.exe ask "what does calc.go do?"   # one question, read-only, no changes
```

Every edit and every command asks you first and shows what it will do. If a task
ends and the answer names a file but nothing was changed or run, BOI says so
under the answer ("Nothing was changed or run in this task.").

## If something does not work

| You see | Likely cause and fix |
|---|---|
| `no valid providers found` | `PSC_1_NAME` and `PSC_1_MODEL` are not set in this shell, or the endpoint is not `localhost` and has no `PSC_1_API_KEY` (any text works for your own server on the network). |
| `Providers: none qualified` in `boi env` | Run `boi provider qualify <name>`. |
| Tools stay off after qualifying | `tool_calling`, `tool_observation` or `authority` did not pass. Try another model or quantization. |
| `timeout` or `budget_exhausted` | Raise `BOI_TASK_TIMEOUT`, `BOI_REQUEST_TIMEOUT` or `BOI_MAX_TOKENS`. |
| The model says it edited a file but did not | See the notice under the answer; the host only counts changes it verified. |

## Measure a model

`bash scripts/acceptance/codingbench.sh MODEL` (Git Bash or WSL on Windows) runs
four small coding tasks and reports passes and false completions. For a local
server: `PSC_1_NAME=ollama PSC_1_BASE_URL=http://localhost:11434/v1 bash
scripts/acceptance/codingbench.sh qwen2.5-coder:7b`.

More: `docs/` (commands, architecture, system flow) and `/help` in the TUI.
