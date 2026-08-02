# Cancellation and Deadlines

This reference documents the runtime's timing and cancellation semantics, which
are deterministic and resolved exactly once per request.

## Caller cancellation

| Stage | Behavior |
| ----- | -------- |
| Before enqueue (`ctx.Err() != nil`) | The request is not enqueued; the context error is returned. |
| While blocked for queue capacity | The blocked submission is removed; the caller returns promptly. |
| Queued before dispatch | The waiter is removed; when it was the last waiter, the unique item is removed and all queue counters are updated. |
| After dispatch | The caller returns promptly with its context error; other waiters in the batch continue. |
| All waiters cancelled | The dedicated batch context is cancelled. |

A short caller never cancels a longer one: individual cancellation only affects
that caller unless every active waiter for the provider work has cancelled.

## Caller deadlines

- Each waiter keeps its own caller context and deadline.
- **Flush timing** uses the earliest active caller deadline minus a configurable
  safety margin, so no caller misses its deadline waiting to batch.
- The **provider batch context** carries the latest active caller deadline, and
  only when every active waiter has a deadline. If any waiter has no deadline,
  the batch context has none (bounded only by engine/scope lifecycle). This
  ensures one short caller cannot cancel longer callers.

## Result versus cancellation race

When a provider result and a caller cancellation are ready at the same time, the
runtime prefers the result: the caller re-checks its result channel after its
context fires and returns the delivered result if present, otherwise the context
error. Delivery happens exactly once; the coordinator marks a waiter delivered
before sending, and a cancelled waiter is skipped.

## Deadline spread

A configurable maximum deadline spread and per-cohort splitting are not exposed
in this release; all compatible items may join one batch subject to the batch
context rules above. This remains a documented compatibility limitation.

## Engine and scope shutdown

`Engine.Close(ctx)` and `Scope.Close(ctx)` respect their context. On abrupt
engine shutdown, remaining active waiters receive `ErrEngineClosed` rather than
blocking forever, and in-flight provider contexts are cancelled.
