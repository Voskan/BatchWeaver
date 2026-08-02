#!/usr/bin/env bash
set -euo pipefail

VERSION="$(tr -d '[:space:]' < release/VERSION)"
EXPECTED_VERSION="v$VERSION"
EXPECTED_REPO="Voskan/BatchWeaver"
EXPECTED_CONFIRM="--confirm-$EXPECTED_VERSION"
GATES="release/gates-$EXPECTED_VERSION.json"
CONFIRM="${1:-}"
DIST="${2:-dist}"

if [ "$CONFIRM" != "$EXPECTED_CONFIRM" ]; then
  printf 'Refusing publication. Pass %s after every gate is reviewed.\n' "$EXPECTED_CONFIRM" >&2
  exit 2
fi

command -v gh >/dev/null
gh auth status >/dev/null
scripts/verify-github-release-gates.sh --publish
test "$(git describe --exact-match --tags HEAD)" = "$EXPECTED_VERSION"
test -f "$DIST/release-manifest.json"
test "$(jq -r .decision "$GATES")" = "ready"
test "$(jq '[.gates[] | select(.required and (.status == "blocked" or .status == "fail"))] | length' "$GATES")" -eq 0
go run ./cmd/batchweaver release verify "$DIST/release-manifest.json"

if gh release view "$EXPECTED_VERSION" --repo "$EXPECTED_REPO" >/dev/null 2>&1; then
  printf 'Release already exists; refusing to overwrite immutable assets.\n' >&2
  exit 3
fi

ASSETS=("$DIST/release-manifest.json" "$DIST/SHA256SUMS")
while IFS= read -r relative; do
  asset="$DIST/$relative"
  if [ ! -f "$asset" ]; then
    printf 'Manifest-declared asset is missing: %s\n' "$relative" >&2
    exit 4
  fi
  ASSETS+=("$asset")
done < <(python3 -c 'import json,sys; print(*[x["path"] for x in json.load(open(sys.argv[1]))["artifacts"]], sep="\n")' "$DIST/release-manifest.json")

gh release create "$EXPECTED_VERSION" "${ASSETS[@]}" \
  --repo "$EXPECTED_REPO" \
  --title "BatchWeaver $EXPECTED_VERSION" \
  --notes-file "docs/release/release-notes-$VERSION.md" \
  --prerelease \
  --verify-tag

gh release view "$EXPECTED_VERSION" --repo "$EXPECTED_REPO" --json url,isPrerelease,tagName
