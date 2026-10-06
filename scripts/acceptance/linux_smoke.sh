#!/usr/bin/env bash
# Linux acceptance for BOI CLI. Run on a Linux machine or VM (e.g. over SSH)
# from the repository root:
#
#   scripts/acceptance/linux_smoke.sh            # offline gates only
#   PSC_1_NAME=... PSC_1_API_KEY=... PSC_1_MODEL=... [PSC_1_BASE_URL=...] \
#     scripts/acceptance/linux_smoke.sh          # also a real Provider round trip
#
# Stage 1-3 need no network beyond Go modules. Stage 4 runs only when a real
# Provider is configured in the environment and makes paid API calls.
set -euo pipefail

repo="$(cd "$(dirname "${BASH_SOURCE[0]}")/../.." && pwd)"
work="$(mktemp -d)"
trap 'rm -rf "$work"' EXIT
bin="$work/boi"

step() { printf '\n== %s\n' "$*"; }
fail() { printf 'FAIL: %s\n' "$*" >&2; exit 1; }

step "1/4 environment"
uname -srm
go version
[ "$(go env GOOS)" = "linux" ] || fail "this script targets Linux"

step "2/4 unit, race and architecture tests"
(cd "$repo" && go vet ./... && go test -race -count=1 ./...)

step "3/4 built-binary folder simulation (fake Provider, no network)"
(cd "$repo" && CGO_ENABLED=0 go build -trimpath -o "$bin" ./cmd/boi)
"$bin" version
python3 -I "$repo/scripts/acceptance/linux_folder_simulation.py" "$bin" >"$work/simulation.json"
if grep -q '"status": "failed"' "$work/simulation.json"; then
	cat "$work/simulation.json"
	fail "folder simulation"
fi
echo "folder simulation: $(grep -c '"status": "passed"' "$work/simulation.json") scenarios passed"

step "4/4 real Provider round trip"
if [ -z "${PSC_1_NAME:-}" ] || [ -z "${PSC_1_API_KEY:-}" ] || [ -z "${PSC_1_MODEL:-}" ]; then
	echo "skipped: set PSC_1_NAME, PSC_1_API_KEY and PSC_1_MODEL to run it"
	echo
	echo "LINUX SMOKE: offline gates passed"
	exit 0
fi
ws="$work/workspace"
mkdir -p "$ws"
cd "$ws"
git init -q
printf 'BOI Linux smoke marker: kite-42\n' >note.txt
"$bin" init </dev/null >/dev/null
"$bin" registry init </dev/null >/dev/null
"$bin" provider qualify "$PSC_1_NAME" </dev/null || fail "provider qualification"

set +e
"$bin" ask --json "Read note.txt and reply with the marker word only." </dev/null >"$work/ask.json"
code=$?
set -e
cat "$work/ask.json"
[ "$code" -eq 0 ] || fail "boi ask exited $code"
grep -q 'kite-42' "$work/ask.json" || fail "answer did not contain the marker from note.txt"
ls agent-folder/output >/dev/null || fail "agent-folder/output missing"

echo
echo "LINUX SMOKE: all gates passed including a real Provider"
