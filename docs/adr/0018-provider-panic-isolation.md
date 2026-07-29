# ADR 0018: Provider and callback panic isolation

- Status: Accepted
- Date: 2026-07-29

## Context

User-provided providers and callbacks may panic. A panic must not corrupt
coordinator state or crash unrelated operations by default.

## Decision

By default the runtime recovers panics: a provider panic becomes a
ProviderPanicError failing the batch; a request-time callback panic becomes a
CallbackPanicError for that request; a hook panic is isolated and reported. A
sanitized value is captured with no stack trace. An engine option can re-panic
for callers who require crash semantics.

## Alternatives considered

- Crash-by-default: rejected for a library runtime.
- Swallowing panics silently: rejected; hides bugs.

## Consequences

The engine stays healthy under misbehaving providers; every callback type is
tested.

## Security and concurrency

Recovered values are sanitized and never include secrets. Provider calls run off
the coordinator goroutine, so a panic cannot corrupt shared state.

## Compatibility

New behavior; configurable.
