# Configuration Reference

BatchWeaver configuration is schema version 1, written in YAML or JSON. Unknown
fields and duplicate keys are errors. See
[../architecture/configuration-pipeline.md](../architecture/configuration-pipeline.md)
for the loading pipeline and [../concepts/operation-contracts.md](../concepts/operation-contracts.md)
for the meaning of operation fields.

## Top level

| Field | Type | Required | Default | Description |
| ----- | ---- | -------- | ------- | ----------- |
| `version` | integer | yes | — | Schema version; must be `1`. |
| `include` | list of strings | no | — | Local include files, relative to this file. |
| `compiler.mode` | enum | no | `transparent` | `transparent` or `disabled`. |
| `runtime.default_scope` | enum | no | `request` | Default batching scope. |
| `security.cross_scope_batching` | bool | no | `false` | Allow cross-scope batching. |
| `security.raw_key_observability` | bool | no | `false` | Expose raw keys in observability. |
| `observability.metrics` | bool | no | `false` | Enable metrics. |
| `observability.tracing` | bool | no | `false` | Enable tracing. |
| `observability.logging` | enum | no | `warnings` | `silent`, `errors`, `warnings`, `info`, `debug`. |
| `operations` | map of ID → operation | no | — | Operation declarations. |
| `extensions` | map of namespace → data | no | — | Preserved, uninterpreted vendor data. |

## Operation

| Field | Type | Required | Description |
| ----- | ---- | -------- | ----------- |
| `kind` | enum | yes | Operation kind (see below). |
| `scalar.symbol` | string | for config-based | Go symbol for the scalar implementation. |
| `batch.symbol` | string | for config-based | Go symbol for the batch implementation. |
| `results.mode` | enum | no | `ordered` (default), `keyed`, `sparse`. |
| `results.missing` | enum | no | `not-found`, `error`, `zero-value`, `contract-violation`. |
| `results.errors` | enum | no | `per-item` (default), `global`, `mixed`. |
| `partition.scope` | enum | no | Batching scope (default `request`). |
| `partition.dimensions` | list | no | Required partition dimensions. |
| `scheduler.mode` | enum | no | Scheduler mode (default `immediate-wave`). |
| `scheduler.min_size` / `max_size` | integer | no | Batch size bounds. |
| `scheduler.max_weight` | integer | no | Aggregate item weight limit. |
| `scheduler.max_payload` / `queue_bytes` | byte size | no | e.g. `4MiB`. |
| `scheduler.max_wait` / `deadline_margin` | duration | no | e.g. `500us`. |
| `scheduler.max_concurrency` | integer | no | Concurrent provider calls. |
| `scheduler.queue_items` | integer | no | Queue item limit. |
| `deduplication.mode` | enum | no | `disabled` (default), `exact`, `canonical`. |
| `deduplication.inflight` / `scope_memoization` | bool | no | Deduplication toggles. |
| `deduplication.max_items` / `max_bytes` | integer / byte size | no | Memoization bounds. |
| `retry.enabled` | bool | no | Enable retry (only where semantics allow). |
| `retry.maximum_attempts` | integer | no | At least 2 when enabled. |
| `retry.initial_backoff` / `maximum_backoff` | duration | no | Backoff bounds. |
| `retry.retryable` | list | no | `transport`, `throttled`, `timeout`, `unavailable`, `conflict`. |
| `fallback.mode` | enum | no | `scalar` (default), `batch-of-one`, `parallel-scalar`, `reject`. |
| `replace` | bool | no | Replace an included operation of the same ID. |
| `extensions` | map | no | Per-operation extension data. |

## Operation kinds

`read-only`, `freshness-sensitive-read`, `idempotent-write`,
`non-idempotent-write`, `commutative-aggregation`, `ordered-aggregation`,
`atomic-group`, `transaction-bound`, `session-bound`.

## Scalar types

- **Durations** use Go syntax with an explicit unit (`0s`, `200us`, `500µs`,
  `5ms`, `1m30s`); bare numbers are rejected.
- **Byte sizes** require a unit: binary (`KiB`, `MiB`, `GiB`, `TiB`) or decimal
  (`KB`, `MB`, `GB`, `TB`); bare numbers are rejected.

## Includes and merge

Includes are applied first in listed order and the including file last.
Operations merge by ID; a duplicate ID is an error unless the later definition
sets `replace: true`. Remote includes are forbidden and absolute includes are
rejected by default.

See the working examples under [examples/configuration](../../examples/configuration).
