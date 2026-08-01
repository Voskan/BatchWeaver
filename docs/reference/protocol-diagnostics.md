# Protocol adapter diagnostics

Network protocol adapters use the `BW7xxx` range, distinct from backend adapters
(`BW6xxx`), proof (`BW5xxx`), runtime (`BW4xxx`), transform (`BW34xx`), and
analysis (`BW3xxx`).

## GraphQL (BW71xx)

| Code | Meaning |
| --- | --- |
| `BW7101` | resolver binding missing |
| `BW7102` | selection-dependent provider requires partitioning |
| `BW7103` | directive creates a batching barrier |
| `BW7104` | field authorization partitions are incompatible |
| `BW7105` | nullability semantics cannot be reconstructed |
| `BW7106` | GraphQL document does not parse |
| `BW7107` | subscription scope would be unbounded |

## gRPC (BW72xx)

| Code | Meaning |
| --- | --- |
| `BW7201` | batch method missing |
| `BW7202` | request key/correlation ambiguous |
| `BW7203` | metadata is incompatible |
| `BW7204` | call option cannot be preserved |
| `BW7205` | per-item status contract missing |
| `BW7206` | stream response lacks a correlation identifier |
| `BW7207` | interceptor semantics unknown |
| `BW7208` | message-size limit would be exceeded |

## HTTP / OpenAPI (BW73xx)

| Code | Meaning |
| --- | --- |
| `BW7301` | batch endpoint is not explicitly declared |
| `BW7302` | item correlation missing |
| `BW7303` | authentication partitions differ |
| `BW7304` | response headers cannot be reconstructed |
| `BW7305` | OpenAPI reference is unsafe |
| `BW7306` | OpenAPI batch extension is invalid |
| `BW7307` | batch size limit exceeded |
