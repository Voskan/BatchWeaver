# From N+1 Calls to Proof-Gated Batching

N+1 access patterns are easy to write and expensive across a network. Manual
batching helps, but a rewrite can alter evaluation order, first-error behavior,
cancellation, transaction identity, or authorization partitions.

BatchWeaver models those obligations explicitly. It discovers a supported
candidate, records evidence, rejects unknowns, creates a deterministic plan,
shows a diff, and runs transformed code through a Go overlay. A typed runtime
coalesces only compatible requests and preserves scalar fallback.

The release supports narrow documented strategies and explicit providers; it does
not batch arbitrary SQL, GraphQL, gRPC, or HTTP. Benchmark results require raw
commands, hardware, Go version, datasets, statistics, and limitations. The release
request is simple: try a sanitized non-production example and report what is
proven, rejected, confusing, or incompatible.
