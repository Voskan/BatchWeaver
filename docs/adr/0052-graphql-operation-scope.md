# ADR 0052: One GraphQL operation per default BatchWeaver scope

- Status: Accepted
- Date: 2026-07-30

## Context

Batching must never cross GraphQL operations, requests, users, or subscription lifetimes.

## Decision

- The default is one BatchWeaver scope per GraphQL operation, derived from the request context, with no cross-operation, cross-request, or subscription-lifetime batching.
- Resolver waves are computed per operation from the execution structure, not textual field order alone.
- Mutations are not reordered; subscriptions use bounded per-event scopes.

## Consequences

Batching stays within a single operation's isolation boundary.
