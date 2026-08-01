# 68. Recursive breadth-first batching only for proven forms

Date: 2026-08-01

## Status

Accepted

## Context

Recursive graph and tree traversals commonly produce N+1 call patterns, but
reordering a traversal can change semantics.

## Decision

Breadth-first, level-batched traversal is offered only for traversals with an
explicit contract (node key, child extraction, terminal condition, limits, cycle
policy, ordering, and error policy) and a valid semantic proof. Breadth-first and
source-child order are preserved; DFS is never silently substituted. Hard depth,
node, edge, and frontier limits return typed errors.

## Consequences

Recursive batching is safe where it applies and refuses to run where the proof is
missing or stale. It is not a general recursion transformer.
