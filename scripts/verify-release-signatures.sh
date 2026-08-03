#!/usr/bin/env bash
# Verify BatchWeaver release signatures and build attestations.
#
# Usage: scripts/verify-release-signatures.sh <artifact-directory> <tag>
#
# Every check is identity-bound: a signature is accepted only when it was
# produced by this repository's signing workflow, from the exact release tag, by
# the expected Sigstore issuer, over the exact artifact digest. A signature that
# is valid in isolation but was produced by a different repository, workflow,
# ref, or identity is rejected.
set -euo pipefail

DIR="${1:-dist}"
TAG="${2:-}"

REPOSITORY="${BATCHWEAVER_REPOSITORY:-Voskan/BatchWeaver}"
WORKFLOW="${BATCHWEAVER_SIGNING_WORKFLOW:-.github/workflows/release-signing.yml}"
ISSUER="${BATCHWEAVER_OIDC_ISSUER:-https://token.actions.githubusercontent.com}"

if [ -z "$TAG" ]; then
  printf 'usage: %s <artifact-directory> <tag>\n' "$0" >&2
  exit 2
fi
if [ ! -d "$DIR" ]; then
  printf 'artifact directory not found: %s\n' "$DIR" >&2
  exit 2
fi
if ! command -v cosign >/dev/null 2>&1; then
  printf 'cosign is required: https://docs.sigstore.dev/cosign/installation/\n' >&2
  exit 2
fi

# The certificate identity is the workflow file at the exact signed tag, so a
# signature made from another ref or another workflow cannot satisfy it.
IDENTITY="https://github.com/${REPOSITORY}/${WORKFLOW}@refs/tags/${TAG}"

verified=0
failed=0

shopt -s nullglob
for artifact in "$DIR"/*.tar.gz "$DIR"/*.zip "$DIR"/*.vsix "$DIR"/SHA256SUMS \
                "$DIR"/*.spdx.json "$DIR"/*.cdx.json "$DIR"/release-manifest.json; do
  base="$(basename "$artifact")"
  sig="$artifact.sig"
  cert="$artifact.pem"

  if [ ! -f "$sig" ] || [ ! -f "$cert" ]; then
    printf 'MISSING  %s (no detached signature or certificate)\n' "$base" >&2
    failed=$((failed + 1))
    continue
  fi

  if cosign verify-blob \
      --signature "$sig" \
      --certificate "$cert" \
      --certificate-identity "$IDENTITY" \
      --certificate-oidc-issuer "$ISSUER" \
      "$artifact" >/dev/null 2>&1; then
    digest="$(sha256sum "$artifact" | cut -d' ' -f1)"
    printf 'OK       %s  sha256:%s\n' "$base" "$digest"
    verified=$((verified + 1))
  else
    printf 'INVALID  %s (identity, issuer, digest, or signature mismatch)\n' "$base" >&2
    failed=$((failed + 1))
  fi
done

if command -v gh >/dev/null 2>&1; then
  printf '\nVerifying GitHub build provenance attestations...\n'
  for artifact in "$DIR"/*.tar.gz "$DIR"/*.zip "$DIR"/*.vsix "$DIR"/release-manifest.json; do
    base="$(basename "$artifact")"
    if gh attestation verify "$artifact" --repo "$REPOSITORY" >/dev/null 2>&1; then
      printf 'OK       %s attestation bound to %s\n' "$base" "$REPOSITORY"
    else
      printf 'INVALID  %s has no attestation from %s\n' "$base" "$REPOSITORY" >&2
      failed=$((failed + 1))
    fi
  done
else
  printf '\ngh is not installed; skipping build-provenance attestation checks.\n' >&2
fi

printf '\nverified=%d failed=%d identity=%s\n' "$verified" "$failed" "$IDENTITY"
if [ "$failed" -ne 0 ] || [ "$verified" -eq 0 ]; then
  printf 'release signature verification FAILED\n' >&2
  exit 1
fi
printf 'release signature verification passed\n'
