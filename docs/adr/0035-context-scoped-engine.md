# ADR 0035: Context-scoped engine resolution

- Status: Accepted
- Date: 2026-07-29

## Context

Generated code must not own a global engine or create an implicit process-wide scope.

## Decision

- The bridge resolves the active scope and bound operation from context.Context.
- With no active scope, Call falls back to the scalar function, preserving behavior.
- The application installs typed bound operations into the scope context; the bridge never constructs a global engine.

## Consequences

No hidden global engines; runtime behavior is explicit and per-scope.
