#!/usr/bin/env bash
set -euo pipefail

EXPECTED_REPO="Voskan/BatchWeaver"
EXPECTED_REMOTE="https://github.com/Voskan/BatchWeaver"

command -v gh >/dev/null
gh auth status >/dev/null
LOGIN=$(gh api user --jq .login)
REPO=$(gh api "repos/$EXPECTED_REPO" --jq .full_name)
REMOTE=$(git remote get-url origin)

test -n "$LOGIN"
test "$REPO" = "$EXPECTED_REPO"
test "$REMOTE" = "$EXPECTED_REMOTE"
test "$(gh api "repos/$EXPECTED_REPO" --jq .visibility)" = "public"
test "$(git status --porcelain --untracked-files=all)" = ""
test -z "$(git tag --list v0.1.0-beta.1)"

gh api "repos/$EXPECTED_REPO/branches/main/protection" >/dev/null
gh api "repos/$EXPECTED_REPO/actions/permissions" >/dev/null
gh api "repos/$EXPECTED_REPO/environments" >/dev/null
gh api "repos/$EXPECTED_REPO/private-vulnerability-reporting" >/dev/null

printf 'Authenticated GitHub release gates passed for %s as %s\n' "$EXPECTED_REPO" "$LOGIN"
