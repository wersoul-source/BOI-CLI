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

body=$(mktemp)
trap 'rm -f "$body"' EXIT
probe() { # $1 = "auth" or "noauth"
	local auth=()
	[ "$1" = auth ] && auth=(-H "Authorization: Bearer $PSC_1_API_KEY")
	curl -sS -m 60 -o "$body" -w '%{http_code}' "$base/chat/completions" "${auth[@]}" \
		-H "Content-Type: application/json" -H "User-Agent: boi-cli-smoke/1.5" \
		-d "{\"model\":\"$model\",\"messages\":[{\"role\":\"user\",\"content\":\"Reply with OK\"}],\"max_tokens\":8}" || echo 000
}

echo "Checking that $model answers on $base ..."
code=$(probe auth)
if [ "$code" != 200 ]; then
	echo "With key: HTTP $code"
	echo "Response: $(head -c 400 "$body" | tr -d '\n')"
	noauth=$(probe noauth)
	echo "Without key: HTTP $noauth"
	echo "Response: $(head -c 400 "$body" | tr -d '\n')"
	case "$code" in
		000) echo "FAIL: cannot reach opencode.ai (network)" >&2 ;;
		401|403) echo "FAIL: OpenCode rejected the request with this key (HTTP $code); see responses above" >&2 ;;
		404) echo "FAIL: model '$model' not found on the chat/completions endpoint" >&2 ;;
		402) echo "FAIL: no credits for '$model'; use a *-free model" >&2 ;;
		*) echo "FAIL: OpenCode returned HTTP $code for '$model'" >&2 ;;
	esac
	exit 1
fi
echo "OK: provider answered"

export PSC_1_NAME=openai PSC_1_BASE_URL="$base" PSC_1_MODEL="$model" PSC_1_API_KEY
make smoke
