# Runtime-lowering diagnostics

Runtime lowering reserves the `BW4xxx` range, distinct from
analysis (`BW3000`, `BW31xx`), transformation (`BW34xx`–`BW38xx`), and proof
(`BW5xxx`).

| Code | Meaning |
| --- | --- |
| `BW4001` | runtime ABI mismatch |
| `BW4002` | generated bridge is stale |
| `BW4101` | scalar fallback would recurse |
| `BW4201` | sibling first-error semantics cannot be preserved |
| `BW4301` | fan-out synchronization pattern is unsupported |
| `BW4302` | errgroup concurrency limit cannot be preserved |
| `BW4401` | required barrier is missing |
| `BW4402` | barrier insertion would change semantics |
| `BW4501` | recursive -toolexec invocation detected |
| `BW4601` | verification comparator unavailable |
| `BW4602` | runtime verification mismatch |

Skip reasons for runtime candidates reuse the transformation skip vocabulary
(unsupported shape, binding unavailable, overlapping region, not eligible).
