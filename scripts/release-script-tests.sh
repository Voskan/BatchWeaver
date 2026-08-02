#!/usr/bin/env bash
set -euo pipefail

REFUSAL_LOG=$(mktemp)
trap 'rm -f "$REFUSAL_LOG"' EXIT
if scripts/publish-prerelease.sh >"$REFUSAL_LOG" 2>&1; then
  printf 'publication helper unexpectedly accepted missing confirmation\n' >&2
  exit 1
fi
rg -q 'Refusing publication' "$REFUSAL_LOG"

test "$(jq -r .decision release/gates-v0.1.0-beta.1.json)" = "blocked"
test "$(jq '[.gates[] | select(.required and (.status == "blocked" or .status == "fail"))] | length' release/gates-v0.1.0-beta.1.json)" -gt 0

printf 'Release helper refusal and blocked-state tests passed.\n'
