# Scan Guide

`batchweaver scan` statically analyzes Go packages and reports potential
batching structure. It never modifies source and never claims a call is safe to
transform.

## Basic usage

```bash
batchweaver scan ./...
```

Example output:

```text
BatchWeaver static analysis

Workspace:               .
Go packages loaded:      103
Application packages:      1
Declared operations:       1
Scalar operation calls:    2
Ambiguous call sites:      0
Invalid declarations:      0

Candidates by structural context:
  loops:              1
  goroutine fan-out:  1
  ...

No source files were changed.
Potential batching structure discovered. Semantic safety has not yet been proven.
```

Counts are repository-dependent.

## Deterministic JSON

```bash
batchweaver scan ./... --format json --reproducible > analysis.json
```

The JSON carries an explicit `schema_version` (`batchweaver.analysis/v1alpha1`),
portable repository-relative paths, and deterministically ordered collections. In
`--reproducible` mode the volatile timestamp is omitted, so output is
byte-identical across runs on the same inputs.

## Flags

| Flag | Meaning |
| ---- | ------- |
| `--format text\|json` | Output format (default text). |
| `--reproducible` | Omit volatile fields for byte-stable output. |
| `--cache-status` | Print local/daemon cache source and hit status to stderr. |
| `--tests` | Include test package variants. |
| `--goos`, `--goarch`, `--tags`, `--cgo` | Build context overrides (reported explicitly). |
| `--fail-on error\|warning\|never` | Nonzero exit threshold (default error). |

Only implemented flags are exposed; SARIF and DOT output are intentionally
absent until those features exist.

## Shared analysis cache

Run `batchweaver daemon start` in a separate terminal to share immutable
analysis snapshots with later CLI and LSP requests for the same workspace. The
daemon is optional; without it, scan uses the same correct in-process analysis.

```bash
batchweaver scan --cache-status ./...
# stderr: analysis cache: source=compute hit=false
batchweaver scan --cache-status ./...
# stderr: analysis cache: source=memory hit=true
```

`daemon status` reports privacy-safe counters and occupancy. Source/config,
build context, toolchain, package pattern, schema, or unsaved-overlay changes
produce a new key. See the [cache policy](../reference/analysis-cache.md).

## Inspect one operation

```bash
batchweaver operation inspect users.get ./...
```

This prints the operation's declaration sources, resolved scalar and batch
symbols, compatibility, discovered call-site count, and structural contexts, and
states that semantic safety has not yet been proven.

## Exit codes

- `0` — analysis completed with no error diagnostics.
- `2` — command-line usage error.
- `3` — analysis produced error diagnostics (or warnings with `--fail-on warning`).

The scan never executes analyzed application code.
