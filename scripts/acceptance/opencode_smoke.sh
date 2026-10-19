#!/usr/bin/env bash
# Run the Linux smoke test against OpenCode Zen with one command:
#
#   bash scripts/acceptance/opencode_smoke.sh
#
# It asks for the API key without echoing it, picks a Zen model that speaks
# the OpenAI chat/completions protocol, and runs `make smoke`. Pass a model
# id as the first argument to skip the automatic choice.
set -euo pipefail

cd "$(dirname "${BASH_SOURCE[0]}")/../.."
base="https://opencode.ai/zen/v1"

if [ -z "${PSC_1_API_KEY:-}" ]; then
	read -r -s -p "OpenCode API key: " PSC_1_API_KEY
	echo
fi
[ -n "$PSC_1_API_KEY" ] || { echo "FAIL: empty API key" >&2; exit 1; }

echo "Checking the key and listing models..."
code=$(curl -sS -m 30 -o /tmp/boi-zen-models.json -w '%{http_code}' \
	"$base/models" -H "Authorization: Bearer $PSC_1_API_KEY") || {
	echo "FAIL: cannot reach opencode.ai (network)" >&2; exit 1; }
case "$code" in
	200) ;;
	401|403) echo "FAIL: OpenCode rejected the API key (HTTP $code)" >&2; exit 1 ;;
	*) echo "FAIL: OpenCode returned HTTP $code" >&2; exit 1 ;;
esac

models=$(grep -o '"id":"[^"]*"' /tmp/boi-zen-models.json | cut -d'"' -f4)
rm -f /tmp/boi-zen-models.json
echo "Models on this key:"
echo "$models" | sed 's/^/  /'

model="${1:-}"
if [ -z "$model" ]; then
	# GPT, Claude and Gemini use other Zen endpoints; BOI speaks chat/completions.
	model=$(echo "$models" | grep -Ei '^(deepseek|glm|kimi|minimax|qwen)' | grep -vi free | head -1 || true)
	[ -n "$model" ] || model=$(echo "$models" | grep -Ei '^(deepseek|glm|kimi|minimax|qwen)' | head -1 || true)
fi
[ -n "$model" ] || { echo "FAIL: no chat/completions model found; pass one as an argument" >&2; exit 1; }
echo "Using model: $model"

export PSC_1_NAME=openai PSC_1_BASE_URL="$base" PSC_1_MODEL="$model" PSC_1_API_KEY
make smoke
