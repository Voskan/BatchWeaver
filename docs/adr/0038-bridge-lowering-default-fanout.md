# ADR 0038: Runtime bridge lowering as the default fan-out strategy

- Status: Accepted
- Date: 2026-07-29

## Context

Sibling, goroutine, and errgroup fan-out call sites need a safe lowering that never adds concurrency.

## Decision

- All runtime-lowering strategies replace the certified scalar call with the same typed bridge call.
- Inside existing goroutine or errgroup fan-out, the bridge coalesces naturally overlapping calls without introducing new concurrency.
- Aggressive parent-level static enqueue and errgroup concurrency-limit-aware coalescing are deferred; the default preserves the original concurrency envelope.

## Consequences

One auditable primitive covers standalone, sibling, and fan-out lowering, and no strategy adds concurrency.
