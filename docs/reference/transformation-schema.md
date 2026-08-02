# Transformation schema

The transformation plan is emitted by `batchweaver transform plan --format=json`.
Its schema version is `batchweaver.transform/v1alpha1`, independent from the
analysis and proof schema versions.

## Plan

| Field | Meaning |
| --- | --- |
| `schema_version` | transformation schema identifier |
| `id` | deterministic plan ID (`bwplan_…`) |
| `workspace` | portable workspace root |
| `toolchain` | Go toolchain identity |
| `build_config` | GOOS, GOARCH, tags, tests |
| `analysis_digest` | analysis inputs the plan was built from |
| `proof_schema` | proof schema of the consumed certificates |
| `contract_digest` | operation contracts in force |
| `strategy_version` | strategy implementation version |
| `transformations` | planned rewrites |
| `files` | per-file digests and line deltas |
| `skipped` | proven candidates not transformed, with reasons |
| `diagnostics` | transformation diagnostics |
| `validation` | parse, type-check, precondition, structural outcomes |
| `digest` | canonical plan digest |

## Transformation

| Field | Meaning |
| --- | --- |
| `id` | transformation ID (`bwtransform_…`) |
| `candidate_id` | analysis candidate ID |
| `certificate_id` | semantic proof certificate ID |
| `strategy` | `static-loop-prefetch` |
| `operation` | operation ID |
| `source` | anchor (file, package, function, range, structural hash, resolution) |
| `phases` | generated phases |
| `generated_symbols` | deterministic generated identifiers |
| `edit_ids` | edits belonging to the transformation |
| `assumptions` | assumptions inherited from the certificate |
| `non_guarantees` | recorded non-guarantees |
| `digest` | canonical transformation digest |

## Source map

The source map (`batchweaver.sourcemap/v1alpha1`) contains segments mapping
generated line ranges to a role, transformation, candidate, and certificate. See
[source maps](../concepts/source-maps.md).

## Determinism

Arrays whose order is not semantically meaningful are sorted by a stable key. IDs
are content-addressed and exclude timestamps and host paths. Plans are
byte-identical across machines for unchanged inputs.
