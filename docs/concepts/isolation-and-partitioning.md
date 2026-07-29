# Isolation and Partitioning

Partitioning prevents semantically or security-incompatible callers from sharing
a batch. In the foundational phase this is a data contract; extraction of
dimension values and runtime enforcement come later.

## Scope

A batch scope bounds where coalescing may occur: `request` (the safe default),
`job`, `graphql-operation`, `session`, `transaction`, `process`, or `custom`.
Process scope is the most dangerous and requires strong isolation.

## Dimensions

A partition contract lists required and optional dimensions. Well-known
dimensions include `receiver`, `tenant`, `authorization`, `transaction`,
`session`, `consistency`, `region`, `encryption-context`, and `deadline-class`.
Custom dimensions must use validated names that do not collide with reserved
ones. Authorization is represented only as a fingerprint dimension, never as a
raw token field.

## Security invariants

Validation enforces, among others:

- process-scope batching requires `tenant` and `authorization` dimensions unless
  the operation is explicitly public and context-independent;
- transaction-bound operations must partition by `transaction`;
- session-bound operations must partition by `session`;
- cross-scope batching cannot be enabled implicitly and, at the configuration
  level, is rejected when `security.cross_scope_batching` is disabled.

These invariants exist so that later runtime enforcement has a safe foundation.
