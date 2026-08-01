# Adapter manifest reference

Schema: `batchweaver.adapter/v1alpha1`.

| Field | Meaning |
| --- | --- |
| `schema_version` | manifest schema identifier |
| `adapter_id` | stable adapter ID (e.g. `database/sql`) |
| `version` | adapter version |
| `status` | `ready` or `deferred` |
| `runtime_abi` | bridge ABI the adapter targets |
| `capabilities` | closed-vocabulary capabilities (implemented only) |
| `dialects` | SQL dialects, when applicable |
| `clients` | client module paths, when applicable |
| `digest` | deterministic content digest |

## Capability vocabulary

`explicit-batch-binding`, `exact-key-read-synthesis`,
`composite-key-read-synthesis`, `ordered-result-mapping`, `keyed-result-mapping`,
`sparse-result-mapping`, `per-item-error`, `global-error`,
`transaction-partitioning`, `session-partitioning`, `prepared-statements`,
`chunking`, `pipeline`, `cluster-slot-partitioning`, `semantic-verification`,
`generated-row-decoding`, `mget`, `hmget`.

A capability appears on a manifest only when implemented and tested. Unknown
capabilities are rejected.
