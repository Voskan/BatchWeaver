#!/usr/bin/env bash
set -euo pipefail

EXPECTED_REPO="Voskan/BatchWeaver"
EXPECTED_REMOTE="https://github.com/Voskan/BatchWeaver"
EXPECTED_TAG="v$(tr -d '[:space:]' < release/VERSION)"
MODE="${1:---pre-tag}"

command -v gh >/dev/null
gh auth status >/dev/null
LOGIN=$(gh api user --jq .login)
REPO=$(gh api "repos/$EXPECTED_REPO" --jq .full_name)
REMOTE=$(git remote get-url origin)

test -n "$LOGIN"
test "$REPO" = "$EXPECTED_REPO"
test "$REMOTE" = "$EXPECTED_REMOTE"
test "$(gh api "repos/$EXPECTED_REPO" --jq .visibility)" = "public"
test "$(gh api "repos/$EXPECTED_REPO" --jq .has_discussions)" = "true"
test "$(gh api "repos/$EXPECTED_REPO/collaborators/$LOGIN/permission" --jq .permission)" = "admin"
test "$(git status --porcelain --untracked-files=all)" = ""

gh api "repos/$EXPECTED_REPO/branches/main/protection" >/dev/null
gh api "repos/$EXPECTED_REPO/actions/permissions" >/dev/null
gh api "repos/$EXPECTED_REPO/environments" >/dev/null
gh api "repos/$EXPECTED_REPO/private-vulnerability-reporting" >/dev/null
gh api "repos/$EXPECTED_REPO/pages" >/dev/null

case "$MODE" in
  --pre-tag)
    test -z "$(git tag --list "$EXPECTED_TAG")"
    test -z "$(git ls-remote --tags origin "refs/tags/$EXPECTED_TAG")"
    if gh release view "$EXPECTED_TAG" --repo "$EXPECTED_REPO" >/dev/null 2>&1; then
      printf 'Release already exists unexpectedly: %s\n' "$EXPECTED_TAG" >&2
      exit 3
    fi
    ;;
  --publish)
    test "$(git describe --exact-match --tags HEAD)" = "$EXPECTED_TAG"
    test "$(git rev-list -n 1 "$EXPECTED_TAG")" = "$(git rev-parse HEAD)"
    test "$(git ls-remote --tags origin "refs/tags/$EXPECTED_TAG^{}" | awk '{print $1}')" = "$(git rev-parse HEAD)"
    ;;
  --repository-only)
    ;;
  *)
    printf 'Usage: %s [--pre-tag|--publish|--repository-only]\n' "$0" >&2
    exit 2
    ;;
esac

printf 'Authenticated GitHub release gates passed for %s as %s (%s)\n' "$EXPECTED_REPO" "$LOGIN" "$MODE"
