# ADR 0019: Runtime event-hook boundary

- Status: Accepted
- Date: 2026-07-29

## Context

Observability is needed without committing to a telemetry backend or leaking
sensitive data.

## Decision

The runtime exposes a backend-neutral `Hooks` struct with an OnEvent callback
receiving typed, redacted `Event` values and an OnError callback. Events carry
only safe metadata (timestamps, IDs, redacted partition tokens, counts, flush
reason, error class). Hooks are invoked outside coordinator event handling where
practical, and hook panics are recovered. No OpenTelemetry dependency is added.

## Alternatives considered

- Direct OpenTelemetry integration now: rejected; premature and heavy.
- Unbounded internal hook queues: rejected; memory risk.

## Consequences

Users can adapt hooks to any backend later; statistics can be disabled to reduce
overhead.

## Security and concurrency

Events never include raw keys or partition components. Hook panics cannot corrupt
runtime state.

## Compatibility

New API; forward-compatible with later observability prompts.
