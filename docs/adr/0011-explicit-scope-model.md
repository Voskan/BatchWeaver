# ADR 0011: Explicit runtime scope model

- Status: Accepted
- Date: 2026-07-29

## Context

Coalescing must be bounded to an explicit region so unrelated work is not merged
and lifetimes are clear. A hidden global scope would be unsafe.

## Decision

A `Scope` is created explicitly (`Engine.NewScope` or `Run`) and carried through
`context.Context` with an unexported key. Calls without a scope return
`ErrScopeRequired` unless a fallback is configured. Nested scopes default to
isolation; reuse and rejection are opt-in. Closing a child never closes a parent.

## Alternatives considered

- Implicit ambient scope: rejected; ambiguous ownership and lifetime.

## Consequences

Batching boundaries are explicit and testable; scope close releases memoization
and detaches from the engine; the context key is not exported.

## Security and concurrency

Scopes bound memoization to a lifetime and prevent cross-scope reuse. Scopes are
safe for concurrent use by goroutines inside them.

## Compatibility

New API; no Prompt 02 changes.
