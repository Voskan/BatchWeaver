#!/usr/bin/env bash
set -euo pipefail

REFUSAL_LOG=$(mktemp)
trap 'rm -f "$REFUSAL_LOG"' EXIT
if scripts/publish-prerelease.sh >"$REFUSAL_LOG" 2>&1; then
  printf 'publication helper unexpectedly accepted missing confirmation\n' >&2
  exit 1
fi
rg -q 'Refusing publication' "$REFUSAL_LOG"

VERSION="$(tr -d '[:space:]' < release/VERSION)"
GATES="release/gates-v$VERSION.json"
test "$(jq -r .decision "$GATES")" = "ready"
test "$(jq '[.gates[] | select(.required and (.status == "blocked" or .status == "fail"))] | length' "$GATES")" -eq 0
rg -q -- 'verify-github-release-gates.sh --publish' scripts/publish-prerelease.sh
rg -q -- 'git ls-remote --tags origin' scripts/verify-github-release-gates.sh

printf 'Release helper refusal and ready-state tests passed.\n'
