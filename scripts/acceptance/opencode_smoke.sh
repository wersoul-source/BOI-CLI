#!/usr/bin/env bash
# Run the Linux smoke test against the OpenCode inference API with one command:
#
#   bash scripts/acceptance/opencode_smoke.sh
#
# It asks for the API key without echoing it, picks a model that speaks
# the OpenAI chat/completions protocol, and runs `make smoke`. Pass a model
# id as the first argument to skip the automatic choice.
set -euo pipefail

cd "$(dirname "${BASH_SOURCE[0]}")/../.."
# OpenCode v2 inference API, OpenAI Chat Completions family:
# POST https://opencode.ai/inference/openai/v1/chat/completions
# Override with OPENCODE_BASE_URL (e.g. the older https://opencode.ai/zen/v1).
base="${OPENCODE_BASE_URL:-https://opencode.ai/inference/openai/v1}"

if [ -z "${PSC_1_API_KEY:-}" ]; then
	read -r -s -p "OpenCode API key: " PSC_1_API_KEY
	echo
fi
[ -n "$PSC_1_API_KEY" ] || { echo "FAIL: empty API key" >&2; exit 1; }

echo "Checking the key and listing models..."
models=""
for list_url in "$base/models" "https://opencode.ai/zen/v1/models"; do
	code=$(curl -sS -m 30 -o /tmp/boi-oc-models.json -w '%{http_code}' \
		"$list_url" -H "Authorization: Bearer $PSC_1_API_KEY") || code=000
	case "$code" in
		200) models=$(grep -o '"id":"[^"]*"' /tmp/boi-oc-models.json | cut -d'"' -f4); break ;;
		401|403) echo "FAIL: OpenCode rejected the API key (HTTP $code)" >&2; rm -f /tmp/boi-oc-models.json; exit 1 ;;
		000) echo "FAIL: cannot reach opencode.ai (network)" >&2; exit 1 ;;
	esac
done
rm -f /tmp/boi-oc-models.json
if [ -n "$models" ]; then
	echo "Models on this key:"
	echo "$models" | sed 's/^/  /'
elif [ -z "${1:-}" ]; then
	echo "FAIL: could not list models; pass a model id, e.g. bash $0 <model-id>" >&2
	exit 1
fi

model="${1:-}"
if [ -z "$model" ]; then
	# GPT, Claude and Gemini are served by other API families; BOI speaks chat/completions.
	model=$(echo "$models" | grep -Ei '(deepseek|glm|kimi|minimax|qwen)' | grep -vi free | head -1 || true)
	[ -n "$model" ] || model=$(echo "$models" | grep -Ei '(deepseek|glm|kimi|minimax|qwen)' | head -1 || true)
fi
[ -n "$model" ] || { echo "FAIL: no chat/completions model found; pass one as an argument" >&2; exit 1; }
echo "Using model: $model"

export PSC_1_NAME=openai PSC_1_BASE_URL="$base" PSC_1_MODEL="$model" PSC_1_API_KEY
make smoke
