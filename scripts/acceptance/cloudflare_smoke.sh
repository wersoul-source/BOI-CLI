#!/usr/bin/env bash
# Run the Linux smoke test against Cloudflare Workers AI:
#
#   bash scripts/acceptance/cloudflare_smoke.sh
#   bash scripts/acceptance/cloudflare_smoke.sh @cf/qwen/qwen2.5-coder-32b-instruct
#
# Workers AI exposes an OpenAI-compatible Chat Completions endpoint at
#   https://api.cloudflare.com/client/v4/accounts/<ACCOUNT_ID>/ai/v1
# authenticated with a Cloudflare API token (Workers AI permission).
# The free allocation is 10,000 Neurons per day; a smoke run uses a small
# fraction of that.
set -euo pipefail

cd "$(dirname "${BASH_SOURCE[0]}")/../.."

model="${1:-@cf/meta/llama-3.3-70b-instruct-fp8-fast}"

# The account ID is the hex segment after dash.cloudflare.com/ in the dashboard URL.
read -r -p "Cloudflare account ID: " account
account="${account//[[:space:]]/}"
[[ "$account" =~ ^[0-9a-f]{32}$ ]] || { echo "FAIL: account ID must be 32 hex characters" >&2; exit 1; }
read -r -s -p "Cloudflare API token (Workers AI): " PSC_1_API_KEY
echo
[ -n "$PSC_1_API_KEY" ] || { echo "FAIL: empty API token" >&2; exit 1; }

base="https://api.cloudflare.com/client/v4/accounts/$account/ai/v1"
body=$(mktemp)
trap 'rm -f "$body"' EXIT

echo "Checking that $model answers on Workers AI ..."
code=$(curl -sS -m 60 -o "$body" -w '%{http_code}' "$base/chat/completions" \
	-H "Authorization: Bearer $PSC_1_API_KEY" -H "Content-Type: application/json" \
	-d "{\"model\":\"$model\",\"messages\":[{\"role\":\"user\",\"content\":\"Reply with OK\"}],\"max_tokens\":8}") || code=000
if [ "$code" != 200 ]; then
	echo "HTTP $code"
	echo "Response: $(head -c 400 "$body" | tr -d '\n')"
	case "$code" in
		000) echo "FAIL: cannot reach api.cloudflare.com (network)" >&2 ;;
		401) echo "FAIL: token rejected; create one with the Workers AI template" >&2 ;;
		403) echo "FAIL: token lacks Workers AI permission, wrong account ID, or model needs Workers Paid" >&2 ;;
		404) echo "FAIL: model '$model' not found" >&2 ;;
		429) echo "FAIL: rate limit or daily free Neurons used up" >&2 ;;
		*) echo "FAIL: Workers AI returned HTTP $code" >&2 ;;
	esac
	exit 1
fi
echo "OK: provider answered"

export PSC_1_NAME=openai PSC_1_BASE_URL="$base" PSC_1_MODEL="$model" PSC_1_API_KEY
make smoke
