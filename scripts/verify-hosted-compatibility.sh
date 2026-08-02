#!/usr/bin/env bash
set -euo pipefail

EXPECTED_REPO="Voskan/BatchWeaver"
COMMIT="${1:-$(git rev-parse HEAD)}"

command -v gh >/dev/null
command -v jq >/dev/null
gh auth status >/dev/null
test "$COMMIT" = "$(git rev-parse "$COMMIT^{commit}")"

RUN_ID=$(gh run list \
  --repo "$EXPECTED_REPO" \
  --workflow compatibility.yml \
  --commit "$COMMIT" \
  --event push \
  --status success \
  --limit 20 \
  --json databaseId,headSha,conclusion \
  --jq ".[] | select(.headSha == \"$COMMIT\" and .conclusion == \"success\") | .databaseId" | head -n 1)

if [ -z "$RUN_ID" ]; then
  printf 'No successful hosted compatibility run exists for %s.\n' "$COMMIT" >&2
  exit 3
fi

ARTIFACT="compatibility-run-$COMMIT"
test "$(gh api "repos/$EXPECTED_REPO/actions/runs/$RUN_ID/artifacts" --jq ".artifacts[] | select(.name == \"$ARTIFACT\" and (.expired | not)) | .name")" = "$ARTIFACT"

BW_COMPAT_TMP=$(mktemp -d)
trap 'rm -rf "$BW_COMPAT_TMP"' EXIT
gh run download "$RUN_ID" --repo "$EXPECTED_REPO" --name "$ARTIFACT" --dir "$BW_COMPAT_TMP"
jq -e --arg commit "$COMMIT" '
  .schema == "batchweaver.compatibility-run/v1alpha1" and
  (.results | length == 18) and
  all(.results[];
    .schema == "batchweaver.compatibility-evidence/v1alpha1" and
    .commit == $commit and .status == "success"
  )
' "$BW_COMPAT_TMP/compatibility-run.json" >/dev/null

printf 'Hosted compatibility evidence verified for %s (run %s).\n' "$COMMIT" "$RUN_ID"
