# Adapter diagnostics

Adapter and SQL-synthesis diagnostics use the `BW6xxx` range, chosen to avoid
collision with analysis (`BW3xxx`), transformation/runtime (`BW34xx`/`BW4xxx`),
and proof (`BW5xxx`). This deviates from the illustrative `BW5xxx` codes in the
prompt to keep every stage's codes distinct.

| Code | Meaning |
| --- | --- |
| `BW6001` | adapter manifest incompatible |
| `BW6002` | adapter capability missing |
| `BW6101` | scalar SQL is dynamic |
| `BW6102` | SQL syntax unsupported |
| `BW6103` | volatile SQL expression / function call |
| `BW6104` | parameter mapping ambiguous |
| `BW6105` | projection mapping ambiguous (`SELECT *`) |
| `BW6106` | result cardinality unsupported |
| `BW6107` | join shape or relation identity unsupported |
| `BW6201` | transaction identity unavailable |
| `BW6202` | generated query exceeds parameter limit |
| `BW6401` | Redis command cannot be safely batched |
| `BW6402` | Redis keys span incompatible cluster slots |
| `BW6501` | adapter contract verification failed |

Each rejection carries a code, human reason, offending node, and byte offset.
