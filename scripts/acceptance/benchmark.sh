#!/usr/bin/env bash
# Measure BOI against a real Provider with a fixed, read-only task set, and
# optionally compare two versions on the same tasks:
#
#   bash scripts/acceptance/benchmark.sh                  # this checkout only
#   BASE_REF=v1.5.0 bash scripts/acceptance/benchmark.sh  # baseline vs this checkout
#   RUNS=3 bash scripts/acceptance/benchmark.sh           # repeat each task
#
# The Provider comes from PSC_1_NAME/PSC_1_API_KEY/PSC_1_MODEL[/PSC_1_BASE_URL].
# When they are not set, the script asks for a Cloudflare Workers AI account
# ID and token. Secrets are read silently and never written to disk.
#
# Per task it records: pass/fail, provider calls, tool calls, steps, tokens
# and wall time. Every task is read-only, so automation mode can run it.
set -euo pipefail

repo="$(cd "$(dirname "${BASH_SOURCE[0]}")/../.." && pwd)"
runs="${RUNS:-1}"
work="$(mktemp -d)"
cleanup() {
	if [ -n "${BASE_REF:-}" ]; then git -C "$repo" worktree remove --force "$work/base-src" >/dev/null 2>&1 || true; fi
	rm -rf "$work"
}
trap cleanup EXIT

if [ -z "${PSC_1_API_KEY:-}" ]; then
	read -r -p "Cloudflare account ID: " account
	account="${account//[[:space:]]/}"
	[[ "$account" =~ ^[0-9a-f]{32}$ ]] || { echo "FAIL: account ID must be 32 hex characters" >&2; exit 1; }
	read -r -s -p "Cloudflare API token (Workers AI): " PSC_1_API_KEY
	echo
	PSC_1_NAME=openai
	PSC_1_BASE_URL="https://api.cloudflare.com/client/v4/accounts/$account/ai/v1"
	PSC_1_MODEL="${PSC_1_MODEL:-@cf/meta/llama-3.3-70b-instruct-fp8-fast}"
fi
[ -n "${PSC_1_NAME:-}" ] && [ -n "${PSC_1_MODEL:-}" ] || { echo "FAIL: set PSC_1_NAME and PSC_1_MODEL" >&2; exit 1; }
export PSC_1_NAME PSC_1_API_KEY PSC_1_MODEL PSC_1_BASE_URL

build() { # $1 = source dir, $2 = output binary
	(cd "$1" && CGO_ENABLED=0 go build -trimpath -o "$2" ./cmd/boi)
}

# Each task: id|prompt|expected substring (case-insensitive)
tasks=(
	"direct|Reply with the word READY only.|READY"
	"single_read|Read note.txt and reply with the marker word only.|kite-42"
	"multi_read|Read a.txt, b.txt and c.txt and reply with the three code words they contain, separated by spaces.|amber birch cedar"
	"search|One file in the docs folder mentions a lantern. Find it and reply with its file name only.|guide.md"
	"recover|Read missing.txt. If it does not exist, reply NOT_FOUND only.|NOT_FOUND"
)

prepare_workspace() { # $1 = dir
	mkdir -p "$1/docs"
	cd "$1"
	git init -q
	printf 'BOI benchmark marker: kite-42\n' >note.txt
	printf 'code word: amber\n' >a.txt
	printf 'code word: birch\n' >b.txt
	printf 'code word: cedar\n' >c.txt
	printf 'Setup notes for the project.\n' >docs/setup.md
	printf 'Carry a lantern when you walk the old path.\n' >docs/guide.md
	printf 'Release history.\n' >docs/changes.md
}

measure() { # $1 = label, $2 = binary
	local label="$1" bin="$2" ws="$work/ws-$1"
	local config="$work/config-$1"
	mkdir -p "$config"
	prepare_workspace "$ws"
	export BOI_CONFIG_DIR="$config"
	"$bin" init </dev/null >/dev/null
	"$bin" registry init </dev/null >/dev/null
	echo "== $label: $("$bin" version | head -1)"
	"$bin" provider qualify "$PSC_1_NAME" </dev/null || { echo "FAIL: $label qualification" >&2; return 1; }
	for entry in "${tasks[@]}"; do
		IFS='|' read -r id prompt expected <<<"$entry"
		for run in $(seq 1 "$runs"); do
			set +e
			"$bin" ask --json "$prompt" </dev/null >"$work/out.json" 2>"$work/err.txt"
			set -e
			python3 -I - "$label" "$id" "$expected" "$work/out.json" >>"$work/results.tsv" <<'PY'
import json, sys
label, task, expected, path = sys.argv[1:5]
try:
    data = json.load(open(path))
except Exception:
    data = {}
usage = data.get("usage") or {}
response = (data.get("response") or "")
ok = data.get("status") == "completed" and all(word.lower() in response.lower() for word in expected.split())
print("\t".join(str(v) for v in [
    label, task, "pass" if ok else "fail", usage.get("provider_calls", 0), usage.get("tool_calls", 0),
    usage.get("steps", 0), usage.get("input_tokens", 0) + usage.get("output_tokens", 0),
    usage.get("duration_ms", 0), data.get("stop_reason") or data.get("status", "error"),
]))
PY
			tail -1 "$work/results.tsv"
		done
	done
	cd "$repo"
}

printf 'version\ttask\tresult\tprovider_calls\ttool_calls\tsteps\ttokens\tduration_ms\tstop\n' >"$work/results.tsv"

if [ -n "${BASE_REF:-}" ]; then
	git -C "$repo" worktree add --detach "$work/base-src" "$BASE_REF" >/dev/null
	build "$work/base-src" "$work/boi-base"
	measure base "$work/boi-base"
fi
build "$repo" "$work/boi-head"
measure head "$work/boi-head"

echo
echo "== summary"
python3 -I - "$work/results.tsv" <<'PY'
import csv, sys
rows = list(csv.DictReader(open(sys.argv[1]), delimiter="\t"))
for version in dict.fromkeys(r["version"] for r in rows):
    group = [r for r in rows if r["version"] == version]
    passed = sum(r["result"] == "pass" for r in group)
    calls = sum(int(r["provider_calls"]) for r in group)
    tokens = sum(int(r["tokens"]) for r in group)
    seconds = sum(int(r["duration_ms"]) for r in group) / 1000
    print(f"{version}: success {passed}/{len(group)} ({100*passed//max(len(group),1)}%), "
          f"provider calls {calls}, tokens {tokens}, time {seconds:.1f}s")
PY
cp "$work/results.tsv" "$repo/benchmark-results.tsv"
echo "rows saved to benchmark-results.tsv"
