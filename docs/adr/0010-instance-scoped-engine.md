# ADR 0010: Instance-scoped runtime engine

- Status: Accepted
- Date: 2026-07-29

## Context

The runtime must own bindings, queues, goroutines, and statistics. A global
registry would create hidden coupling and reintroduce the init-time registration
that ADR 0008 rejected.

## Decision

All runtime state lives on an explicit `Engine` instance created with
`NewEngine`. Bindings, scopes, queues, and providers belong to an engine; context
helpers carry pointers to explicit instances but never resolve them through
global state. There are no package-level mutable maps.

## Alternatives considered

- A process-global default engine: rejected; it hides ownership and complicates
  testing and shutdown.

## Consequences

Multiple independent engines can coexist; tests construct isolated engines;
shutdown is well defined.

## Security and concurrency

No shared global mutable state removes a class of data races and cross-tenant
leaks. The engine is safe for concurrent use.

## Compatibility

Preserves the Prompt 02 no-global-registration decision.
