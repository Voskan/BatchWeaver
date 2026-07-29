# Batching Model

This document defines the vocabulary BatchWeaver uses. The terms are stable
concepts even though the mechanisms that implement them are not yet built.

- **Scalar operation** — an operation invoked once per item, handling a single
  logical request (for example, loading one record by id).

- **Batch operation** — an operation that handles many logical requests in a
  single invocation (for example, loading many records by a list of ids).

- **Logical request** — one unit of intended work from the caller's point of
  view, independent of whether it is executed alone or as part of a batch.

- **Batch scope** — a bounded region of execution within which logical requests
  may be coalesced. Scopes define where and for how long coalescing is allowed.

- **Partition** — a grouping of logical requests that must be batched together
  (or kept apart), for example by shard, tenant, or connection.

- **Deduplication** — collapsing identical in-flight logical requests so the
  underlying work is performed once and the result is shared.

- **Batching barrier** — a point that prevents coalescing across it, because
  doing so would change observable behavior (for example, an ordering or
  side-effect dependency).

- **Semantic transparency** — the guarantee that batched execution produces the
  same observable behavior as the original scalar execution.

- **Native batch** — a backend-provided batch operation that BatchWeaver targets
  directly, as opposed to one synthesized by grouping scalar calls.

- **Runtime coalescing** — combining independent logical requests into batches at
  execution time, based on what is in flight within a scope.

- **Static batching** — transforming code at or before build time so that
  batching structure is determined statically rather than purely at runtime.

- **Batch wave** — a set of logical requests that are dispatched together as one
  batch, typically formed when a scope flushes.
