#!/usr/bin/env bash
set -euo pipefail

OUTPUT="${1:-artifacts/benchmarks}"
case "$OUTPUT" in ""|"/"|"."|"..") printf 'unsafe benchmark output path\n' >&2; exit 2;; esac
mkdir -p "$OUTPUT"
go version > "$OUTPUT/environment.txt"
uname -a >> "$OUTPUT/environment.txt"
git rev-parse HEAD >> "$OUTPUT/environment.txt"
go test ./runtime/... -run '^$' -bench . -benchmem -count 10 > "$OUTPUT/runtime.txt"
go test ./internal/assurance -run '^TestPerformanceBudget' -count 1 > "$OUTPUT/release-budget.txt"
printf 'Raw benchmark evidence written to %s; use benchstat and report hardware, dataset, confidence, and limitations before publication.\n' "$OUTPUT"
