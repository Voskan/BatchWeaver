# Network adapter limitations (this stage)

This stage delivers the network protocol adapter SDK, a fully implemented HTTP
batch adapter, GraphQL resolver-wave analysis, gRPC batch-binding and metadata
policy, OpenAPI extension loading, and protocol verification — all dependency-free
and hermetically tested.

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
- CLI: `adapter list --category`, `graphql inspect|graph`, `grpc inspect`,
  `http verify`, `openapi validate|inspect`.

## Deferred (blocked offline; contracts ready)

- The concrete **gqlgen** runtime hooks and **grpc-go/bufconn** client and server
  integrations are not compiled in because their dependency closures are
  unavailable with the module proxy disabled. Their manifests are marked
  `deferred`, capabilities defined, and the framework-neutral logic (GraphQL model
  and waves, gRPC binding/metadata/status policy) implemented so the integrations
  are thin additions once the dependencies are available.

## Not implemented (out of scope)

- Universal GraphQL query optimization, arbitrary GraphQL-to-SQL, cross-operation
  or cross-request GraphQL merging, and subscription-lifetime batching.
- Automatic server-side generation of remote gRPC batch methods.
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
