# ADR 0053: gqlgen integration through supported extension APIs

- Status: Accepted
- Date: 2026-07-30

## Context

BatchWeaver must integrate with gqlgen without patching its internals.

## Decision

- The GraphQL adapter keeps a framework-neutral operation model and resolver-wave analysis, independent of any framework.
- Concrete gqlgen integration is through its supported extension, middleware, and field-context hooks only; gqlgen internals are never patched.
- The concrete gqlgen runtime hook is deferred in this build because the dependency is unavailable offline; the neutral model and wave analysis are implemented and tested.

## Consequences

The GraphQL logic is reusable and testable now; the gqlgen binding is a thin, well-scoped addition.
