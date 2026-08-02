# ADR 0016: Deterministic scheduler and clock abstraction

- Status: Accepted
- Date: 2026-07-29

## Context

Scheduling must be testable without real sleeps and must not busy-wait. Adaptive
scheduling is out of scope for this prompt.

## Decision

The scheduler supports immediate, fixed-window, deadline-aware, and manual modes,
driven by a single timer per operation over a `Clock` abstraction. Production
uses a system clock; tests use `testing/synctest` (and a controllable clock) for
deterministic timing. Adaptive, throughput, and latency spec modes are treated as
a fixed window (documented). Wave semantics are provided through explicit Flush.

## Alternatives considered

- Real-time sleeps in tests: rejected; flaky and slow.
- Per-partition Go timers: rejected; one coalesced timer is simpler and correct.

## Consequences

Deterministic scheduler tests; no busy-waiting; adaptive learning deferred.

## Security and concurrency

Timers are owned by the coordinator and stopped on shutdown; no goroutine per
request or per idle partition.

## Compatibility

Uses the scheduler policy types.
