// Package runtime is the reserved home for BatchWeaver's execution-time
// batching support.
//
// No runtime behavior is implemented in the repository bootstrap. Later prompts
// will introduce, incrementally and only when each is real and tested:
//
//   - batch scopes that delimit where coalescing may occur;
//   - request coalescing of independent logical requests into batches;
//   - typed operation registries mapping scalar operations to batch operations;
//   - scheduling of batch waves;
//   - deduplication of identical in-flight requests;
//   - partitioning of requests across batch keys or shards;
//   - context propagation and cancellation across coalesced work;
//   - distribution of batch results back to individual callers;
//   - metrics and tracing hooks for observability.
//
// This package deliberately contains no placeholder queues or schedulers. To
// preserve a clean dependency direction, it must never import BatchWeaver's
// compiler or analysis packages; generated code will eventually depend on a
// minimal, stable subset of this package.
package runtime
