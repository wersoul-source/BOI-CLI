# Get Started with BOI CLI

The short version is in the [README](README.md#quick-start): install, run
`boi quickstart` in your project, then run `boi`. This page adds what to do for a
slow local model, what each step does, and what to check when something is off.

## The three steps

**1. Install** (Linux, or Windows in PowerShell) from the README, or build from
source (Go 1.24 or later):

```powershell
git clone https://github.com/wersoul-source/BOI-CLI.git
cd BOI-CLI
go build -o boi.exe ./cmd/boi        # Linux: go build -o boi ./cmd/boi
```

**2. Get ready.** In the folder of your project:

```powershell
boi quickstart
```

It looks for a model server on this computer (Ollama, LM Studio, llama.cpp),
lists the models it finds and lets you pick one with Enter. With none running,
it offers a hosted provider with an API key, or a server at another address. It
then saves the choice in `.env` (kept out of Git) and checks what the model can
do. Run it again whenever you change the model.

For a hosted provider the wizard lists OpenAI, Anthropic, Google, Groq,
OpenRouter and others: pick one with the arrows, paste the key (it is hidden),
pick or type the model. **A key only works on its own provider's server.** For
anything not in the list, choose **Other** at the bottom and give the address:

| Provider | Address to give |
|---|---|
| Cloudflare Workers AI | `https://api.cloudflare.com/client/v4/accounts/<ACCOUNT_ID>/ai/v1`, with a model such as `@cf/qwen/qwen2.5-coder-32b-instruct` |
| A server on your network | `http://192.168.1.20:8080/v1` |

**3. Use it.**

```powershell
boi                                      # chat and work on the project
boi ask "what does calc.go do?"          # one question, read-only
```

Every edit and every command asks you first and shows what it will do. If a task
ends and the answer names a file but nothing was changed or run, BOI says so
under the answer ("Nothing was changed or run in this task.").

## No model yet?

The quickest free start is [Ollama](https://ollama.com): install it, then

```powershell
ollama pull qwen2.5-coder:7b
```

and run `boi quickstart`. Use the model's exact name if you pull another one.

## A slow local model (for example 7B on 6 GB of video memory)

Give the model the context you want when you start the server, not in BOI:
llama.cpp `-c 65536`; Ollama `OLLAMA_CONTEXT_LENGTH=65536` before `ollama serve`
(or `PARAMETER num_ctx 65536` in a Modelfile); LM Studio, the context-length
setting. A 7B model with a 64k context does not fit entirely in 6 GB, so part of
it runs on the CPU and answers are slow.

The defaults suit a hosted API: 2 minutes per task and per request, 64,000 tokens
per task. Raise them before you start `boi`:

```powershell
$env:BOI_TASK_TIMEOUT    = "15m"       # a whole task (10s to 2h)
$env:BOI_REQUEST_TIMEOUT = "10m"       # one request to the model (10s to 2h)
$env:BOI_MAX_TOKENS      = "200000"     # input plus output over the whole task
# also: BOI_MAX_TOOL_CALLS (default 8), BOI_MAX_STEPS (default 12)
```

A value that is not valid stops start-up and names the variable. The first check
of a local model gets 3 minutes per step, because its first answer loads the
model.

## What the check means

`boi quickstart` (or `boi provider qualify <name>`) runs short probes and keeps
only what the model proved:

- `completion`, `reasoning`, `context` passed: it can chat.
- `tool_calling`, `tool_observation`, `authority` passed: the Agent may read, edit
  and run commands (each with your approval).
- `native_tool_schema` failed: normal. BOI then talks to the model in plain text.
- `skill_calling` passed: Skills are offered.

A capability that failed stays off, and the Agent then answers without touching
files. Run the check again after you change the model or endpoint, or after 30
days.

## By hand (what quickstart does)

Without the wizard, a local server needs no API key. In PowerShell:

```powershell
$env:PSC_1_NAME     = "ollama"
$env:PSC_1_BASE_URL = "http://localhost:11434/v1"   # llama.cpp: :8080/v1   LM Studio: :1234/v1
$env:PSC_1_MODEL    = "qwen2.5-coder:7b"
boi provider qualify ollama
```

The same four `PSC_1_*` lines work in a `.env` file in the project folder, which
the Agent can never read. `boi setup` is the full-screen wizard for hosted
providers. `boi env` shows what the Agent can use; `boi env presets` lists
read-only and coding setups.

## If something does not work

| You see | Likely cause and fix |
|---|---|
| `boi: command not found` / not recognized | Open a new terminal after installing, or run `./boi` from the folder it was built in. |
| Every check says `HTTP 401` | The server did not accept the key: the key belongs to another provider than the address shown on the `Endpoint:` line, or was copied incompletely. Run `boi quickstart` again, and for a provider that is not in the list choose **Other** and give its address. Create a new key if the old one may have been sent to the wrong server. |
| `quickstart` finds no local model | The server is not running, or it is on another port: choose "A local server at another address". |
| `no valid providers found` | `PSC_1_NAME` and `PSC_1_MODEL` are not set in this shell, or the endpoint is not `localhost` and has no `PSC_1_API_KEY` (any text works for your own server on the network). |
| `Providers: none qualified` in `boi env` | Run `boi quickstart` again. |
| The Agent only chats | The model did not pass `tool_calling`, `tool_observation` or `authority`. Try a larger or coder-tuned model. |
| `timeout` or `budget_exhausted` | Raise `BOI_TASK_TIMEOUT`, `BOI_REQUEST_TIMEOUT` or `BOI_MAX_TOKENS`. |
| The model says it edited a file but did not | Read the notice under the answer; BOI only counts changes it verified. |

## Measure a model

`bash scripts/acceptance/codingbench.sh MODEL` (Git Bash or WSL on Windows) runs
four small coding tasks and reports passes and false completions. For a local
server: `PSC_1_NAME=ollama PSC_1_BASE_URL=http://localhost:11434/v1 bash
scripts/acceptance/codingbench.sh qwen2.5-coder:7b`.

More: `docs/` (commands, architecture, system flow) and `/help` in the TUI.
