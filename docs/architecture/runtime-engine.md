# Runtime Engine

The runtime (`runtime`, imported as `batchruntime`) is BatchWeaver's explicit,
typed request-coalescing execution layer. Applications may invoke it through
typed handles, and proven compiler transformations target it through `bridge`.

## Ownership

- An **Engine** owns bound operations, their coordinator goroutines, lifecycle
  cancellation, and aggregate statistics. There is no global registry; every
  binding belongs to an explicit engine.
- A **Scope** owns its identity, lifecycle, scope-local memoization, and the set
  of operations used within it. Scopes are carried through `context.Context`.
- A **BoundOperation** owns the immutable operation spec, provider, key strategy,
  partitioner, queue policy, and optional fallback.
- A **logical request** owns its request ID, cloned key material, a single
  waiter, and its own completion state.

## Concurrency model

Each bound operation runs exactly one **coordinator goroutine** that owns all of
its mutable state (partition queues, the in-flight deduplication index, timers)
and processes submissions, cancellations, completions, and flush/drain/close
control **serially**. Because only that goroutine touches the state, the core
needs no locks.

- User callbacks (partitioner, key clone/hash/size, weight) run in the **caller**
  goroutine before submission, never inside the coordinator.
- Provider calls run on **bounded worker goroutines**, never while the
  coordinator is processing an event, limited by the per-operation concurrency.
- Callers wait on their own buffered result channel and select on it against
  their context, so one caller's cancellation or deadline never affects another.

```text
caller goroutine ──submit──▶ coordinator goroutine ──dispatch──▶ worker goroutine ──▶ provider
      ▲                          (owns queues,                    (bounded by            │
      └────── result channel ──── dedup, timers) ◀── complete ─── maxConcurrency) ◀──────┘
```

## Lifecycle

```text
Engine:  open ─▶ closing ─▶ closed
Scope:   open ─▶ closed
```

Binding on a non-open engine fails. Submissions after `closing` are rejected (or
fall back if configured). `Engine.Close(ctx)` drains in-flight work bounded by
`ctx`, cancels the engine context, waits for coordinator and worker goroutines,
and is idempotent. Every goroutine has an owner and a termination path; every
timer is owned by its coordinator and stopped on shutdown.

## Deadline algorithm

The provider batch context is dedicated, not borrowed from one caller. It is
cancelled when the engine or scope closes and when **every** active waiter in the
batch has cancelled. It carries the **latest** active caller deadline only when
**every** active waiter has a deadline, so a short caller can never cancel a
longer one. Flush timing uses the **earliest** active caller deadline minus a
safety margin. See [../reference/cancellation-and-deadlines.md](../reference/cancellation-and-deadlines.md).

## Result validation

Provider responses are validated against the exact batch request IDs before any
value reaches a caller. Missing, duplicate, or unexpected results are turned into
a deterministic contract error for the batch; per-item errors preserve
`errors.Is`/`errors.As`. See [../reference/provider-contract.md](../reference/provider-contract.md).

## Deduplication and memoization

In-flight deduplication collapses overlapping same-key work within a partition;
scope memoization reuses a completed read result later within a scope. They are
separate layers. See [../concepts/deduplication.md](../concepts/deduplication.md).

## Compiler integration

The compiler discovers eligible scalar call sites and rewrites proven candidates
through a typed `bridge.Operation`. The bridge resolves the bound operation and
uses the same reflection-free `BoundOperation.Do` path described here, with a
direct scalar fallback when no compatible binding is available.
