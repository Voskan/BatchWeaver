# Network Adapter Limitations

This build delivers the network protocol adapter SDK, HTTP batching, GraphQL
resolver-wave analysis, a concrete gqlgen extension, explicit grpc-go unary
batch invocation, OpenAPI loading, and protocol verification.

## Supported

- Protocol adapter manifests and capabilities (network category) with deterministic
  digests.
- **HTTP/OpenAPI (ready):** explicit batch endpoints over `net/http` with typed
  keyed/positional JSON envelopes, correlation validation (missing/duplicate),
  deterministic chunking, transport identity preservation, `x-batchweaver` OpenAPI
  3.1+ loading (JSON + YAML) with bounded size and no remote refs, and hermetic
  `httptest` contract verification.
- **GraphQL:** a framework-neutral operation model, a recursive-descent query
  parser (no regex, never panics), resolver-wave computation, and normalized
  selection digests.
- **gRPC:** explicit unary batch-binding model with strict validation, metadata
  partition policy, response-correlation modes, and a streaming lifecycle/state
  vocabulary.
- **gqlgen:** public operation/field interceptors establish one runtime scope per
  operation and expose normalized field partitions without changing results.
- **grpc-go:** a typed explicit unary provider is tested over a real bufconn
  transport, including exact response-ID validation and metadata partitioning.
- CLI: `adapter list --category`, `graphql inspect|graph`, `grpc inspect`,
  `http verify`, `openapi validate|inspect`.

## Not implemented (out of scope)

- Universal GraphQL query optimization, arbitrary GraphQL-to-SQL, cross-operation
  or cross-request GraphQL merging, and subscription-lifetime batching.
- Automatic server-side generation of remote gRPC batch methods and automatic
  batching of streaming RPCs.
- Arbitrary HTTP request fusion, undocumented batch protocols, NDJSON/multipart
  batch bodies (capability reserved only), and remote OpenAPI reference resolution.
- Distributed cross-process batching and any deployment/gateway concern.

## Diagnostics

Network adapter codes use the `BW7xxx` range (BW71xx GraphQL, BW72xx gRPC, BW73xx
HTTP/OpenAPI), distinct from the backend adapters' `BW6xxx`.

## Non-guarantees

Transport multiplexing is never treated as application batching. One caller's
authorization metadata is never used for another caller. Per-item results are
never invented from a batch-level failure. Unsupported protocol shapes are
rejected conservatively with an exact diagnostic.
