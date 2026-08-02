#!/usr/bin/env bash
set -euo pipefail

SOURCE_ROOT="$(git rev-parse --show-toplevel)"
SOURCE_COMMIT="$(git -C "$SOURCE_ROOT" rev-parse HEAD)"
RUN_DIR="$(mktemp -d "${TMPDIR:-/tmp}/batchweaver-release-XXXXXXXX")"
cleanup() {
  case "$RUN_DIR" in
    "${TMPDIR:-/tmp}"/batchweaver-release-*) rm -rf -- "$RUN_DIR" ;;
    *) printf 'refusing unsafe cleanup path: %s\n' "$RUN_DIR" >&2 ;;
  esac
}
trap cleanup EXIT INT TERM

git clone --no-local --no-checkout "$SOURCE_ROOT" "$RUN_DIR/source"
git -C "$RUN_DIR/source" checkout --detach "$SOURCE_COMMIT"
test -z "$(git -C "$RUN_DIR/source" status --porcelain --untracked-files=all)"

cd "$RUN_DIR/source"
go generate ./...
git diff --exit-code
make check

RELEASE_PATH="$PATH"
if (( "$(node -p 'Number(process.versions.node.split(".")[0])')" < 22 )); then
  NODE22_BIN="$(npm exec --yes --package=node@22 -- node -p 'process.execPath')"
  RELEASE_PATH="$(dirname "$NODE22_BIN"):$PATH"
fi
PATH="$RELEASE_PATH" go run ./cmd/batchweaver release build --snapshot --output "$RUN_DIR/dist-first"
PATH="$RELEASE_PATH" go run ./cmd/batchweaver release verify "$RUN_DIR/dist-first/release-manifest.json"
PATH="$RELEASE_PATH" go run ./cmd/batchweaver release reproduce --manifest "$RUN_DIR/dist-first/release-manifest.json"

mkdir -p "$SOURCE_ROOT/dist"
cp -R "$RUN_DIR/dist-first/." "$SOURCE_ROOT/dist/"
printf 'Clean-checkout unpublished release dry run passed at %s\n' "$SOURCE_COMMIT"
