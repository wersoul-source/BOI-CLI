#!/usr/bin/env bash
# Run the coding benchmark (scripts/acceptance/codingbench) against one or
# more Cloudflare Workers AI models, asking once for the account ID and token:
#
#   bash scripts/acceptance/codingbench.sh                       # default model
#   bash scripts/acceptance/codingbench.sh @cf/qwen/qwen2.5-coder-32b-instruct
#   RUNS=3 bash scripts/acceptance/codingbench.sh MODEL_A MODEL_B   # compare
#
# Another Provider: set PSC_1_NAME and PSC_1_MODEL (and PSC_1_BASE_URL, and
# PSC_1_API_KEY unless the server is local) and the script uses them, for
# example a local Ollama:
#   PSC_1_NAME=ollama PSC_1_BASE_URL=http://localhost:11434/v1 bash scripts/acceptance/codingbench.sh qwen2.5-coder:7b
# A slow local model needs longer limits: BOI_TASK_TIMEOUT=15m BOI_REQUEST_TIMEOUT=10m
# (the benchmark's own per-task limit then follows BOI_TASK_TIMEOUT).
#
# CF_ACCOUNT_ID skips the account question. The token is read silently, kept in
# memory only, and never printed.
set -euo pipefail

repo="$(cd "$(dirname "${BASH_SOURCE[0]}")/../.." && pwd)"
runs="${RUNS:-3}"
default_model="@cf/meta/llama-3.3-70b-instruct-fp8-fast"
models=("$@")
[ ${#models[@]} -gt 0 ] || models=("${PSC_1_MODEL:-$default_model}")

if [ -z "${PSC_1_API_KEY:-}" ] && [ -z "${PSC_1_NAME:-}" ]; then
	account="${CF_ACCOUNT_ID:-}"
	if [ -z "$account" ]; then
		read -r -p "Cloudflare account ID: " account
	fi
	account="${account//[[:space:]]/}"
	[[ "$account" =~ ^[0-9a-f]{32}$ ]] || { echo "FAIL: account ID must be 32 hex characters" >&2; exit 1; }
	read -r -s -p "Cloudflare API token (Workers AI): " PSC_1_API_KEY
	echo
	[ -n "$PSC_1_API_KEY" ] || { echo "FAIL: the token is empty" >&2; exit 1; }
	PSC_1_NAME=openai
	PSC_1_BASE_URL="https://api.cloudflare.com/client/v4/accounts/$account/ai/v1"
fi
[ -n "${PSC_1_NAME:-}" ] || { echo "FAIL: set PSC_1_NAME (or leave PSC_1_API_KEY unset to use Cloudflare)" >&2; exit 1; }
export PSC_1_NAME PSC_1_BASE_URL
export PSC_1_API_KEY="${PSC_1_API_KEY:-}"

work="$(mktemp -d)"
trap 'rm -rf "$work"' EXIT
(cd "$repo" && go build -o "$work/codingbench" ./scripts/acceptance/codingbench)

summary=()
for model in "${models[@]}"; do
	echo
	echo "=== $model (runs per task: $runs)"
	PSC_1_MODEL="$model" "$work/codingbench" -runs "$runs" 2>&1 | tee "$work/out.txt" || true
	summary+=("$model: $(grep -m1 '^success' "$work/out.txt" || echo 'no result (see probe lines above)')")
done

echo
echo "=== summary"
printf '%s\n' "${summary[@]}"
