# Proof diagnostics

The proof engine reserves the `BW5xxx` diagnostic range. A proven-eligible
candidate produces no diagnostic. A proven-ineligible candidate produces a
warning; unknown and assumption-required candidates produce informational
diagnostics, so an unknown is never reported as an error.

## Code families

| Code | Family | Meaning |
| --- | --- | --- |
| `BW5000` | declaration | signature, enablement, or result-contract issue |
| `BW5100` | order | evaluation-order or barrier conflict |
| `BW5200` | dependency | target resolution or data-dependency conflict |
| `BW5300` | effect | operation-category or effect conflict |
| `BW5400` | receiver/key | receiver or key stability conflict |
| `BW5500` | result/error | result or error reconstruction conflict |
| `BW5600` | context | context, deadline, or cancellation conflict |
| `BW5700` | transaction | transaction, session, or consistency conflict |
| `BW5800` | panic | panic, defer, or recover conflict |
| `BW5900` | precision | assumption, precision, or resource-limit issue |

Each diagnostic carries a stable code and fingerprint, a severity, the candidate
and (where applicable) strategy, the primary source location, the blocking
obligation ID, and a short message. Diagnostics never claim that an automatic fix
exists.

## Reason codes

Non-eligible decisions carry a machine-readable `reason_code`, including
`ambiguous_target`, `observable_barrier`, `write_category`, `missing_contract`,
`invalid_declaration`, `disabled_operation`, and `resource_limit`.
