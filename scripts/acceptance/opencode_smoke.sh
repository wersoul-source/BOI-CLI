#!/usr/bin/env bash
# Run the Linux smoke test against the OpenCode Console inference API:
#
#   bash scripts/acceptance/opencode_smoke.sh            # free default model
#   bash scripts/acceptance/opencode_smoke.sh glm-5.3    # any chat/completions model
#
# BOI speaks the OpenAI Chat Completions protocol, so the model must be one
# the OpenCode docs list under /inference/openai/v1/chat/completions
# (DeepSeek, GLM, Kimi, MiniMax, Mistral, Qwen3.8 Max and the *-free models).
# GPT, Claude, Gemini and most Qwen models use other endpoints and will fail.
set -euo pipefail

cd "$(dirname "${BASH_SOURCE[0]}")/../.."

# Free, zero-retention, Chat Completions model from the OpenCode Console docs.
model="${1:-deepseek-v4.1-flash-free}"
base="https://opencode.ai/inference/openai/v1"

# Always ask: a PSC_1_API_KEY left over in the shell is a common cause of 401/403.
read -r -s -p "OpenCode API key: " PSC_1_API_KEY
echo
[ -n "$PSC_1_API_KEY" ] || { echo "FAIL: empty API key" >&2; exit 1; }

echo "Checking that $model answers on $base ..."
code=$(curl -sS -m 60 -o /dev/null -w '%{http_code}' "$base/chat/completions" \
	-H "Authorization: Bearer $PSC_1_API_KEY" -H "Content-Type: application/json" \
	-d "{\"model\":\"$model\",\"messages\":[{\"role\":\"user\",\"content\":\"Reply with OK\"}],\"max_tokens\":8}") || code=000
case "$code" in
	200) echo "OK: provider answered" ;;
	000) echo "FAIL: cannot reach opencode.ai (network)" >&2; exit 1 ;;
	401|403) echo "FAIL: OpenCode rejected the API key (HTTP $code)" >&2; exit 1 ;;
	404) echo "FAIL: model '$model' not found on the chat/completions endpoint (HTTP 404)" >&2; exit 1 ;;
	402) echo "FAIL: no credits for '$model' (HTTP 402); use a *-free model" >&2; exit 1 ;;
	*) echo "FAIL: OpenCode returned HTTP $code for '$model'" >&2; exit 1 ;;
esac

export PSC_1_NAME=openai PSC_1_BASE_URL="$base" PSC_1_MODEL="$model" PSC_1_API_KEY
make smoke
