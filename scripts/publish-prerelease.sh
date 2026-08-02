#!/usr/bin/env bash
set -euo pipefail

EXPECTED_VERSION="v0.1.0-beta.1"
EXPECTED_REPO="Voskan/BatchWeaver"
CONFIRM="${1:-}"
DIST="${2:-dist}"

if [ "$CONFIRM" != "--confirm-v0.1.0-beta.1" ]; then
  printf 'Refusing publication. Pass --confirm-v0.1.0-beta.1 after every gate is reviewed.\n' >&2
  exit 2
fi

scripts/verify-github-release-gates.sh
test "$(git describe --exact-match --tags HEAD)" = "$EXPECTED_VERSION"
test -f "$DIST/release-manifest.json"
go run ./cmd/batchweaver release verify "$DIST/release-manifest.json"

if gh release view "$EXPECTED_VERSION" --repo "$EXPECTED_REPO" >/dev/null 2>&1; then
  printf 'Release already exists; refusing to overwrite immutable assets.\n' >&2
  exit 3
fi

ASSETS=()
for asset in "$DIST"/* "$DIST"/reports/*; do
  if [ -f "$asset" ]; then ASSETS+=("$asset"); fi
done
test "${#ASSETS[@]}" -gt 0

gh release create "$EXPECTED_VERSION" "${ASSETS[@]}" \
  --repo "$EXPECTED_REPO" \
  --title "BatchWeaver v0.1.0-beta.1" \
  --notes-file docs/release/release-notes-0.1.0-beta.1.md \
  --prerelease \
  --verify-tag

gh release view "$EXPECTED_VERSION" --repo "$EXPECTED_REPO" --json url,isPrerelease,tagName
