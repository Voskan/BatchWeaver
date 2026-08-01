# Recursive breadth-first batching

Recursive graph and tree traversals are a common source of N+1 backend calls.
For traversals with an explicit contract and a valid semantic proof, BatchWeaver
loads the graph breadth-first: one batched backend call per frontier level.

## Contract

A recursive traversal declares its node key, child extraction, terminal
condition, hard limits (depth, nodes, edges, frontier), cycle policy, ordering,
and error policy. `ProofValid` must be true; a stale or missing proof yields
`BW8103` and refuses to run.

## Semantics preserved

Breadth-first order and source child order are preserved; DFS is never silently
substituted. Duplicate handling within a frontier is stable. Cycle handling is
explicit: `error`, `skip-seen`, or `return-cycle-marker`. Error handling is
explicit: `fail-fast`, `collect-per-node`, or `partial-graph`. Cancellation and
deadlines are honored through the context.

## Limits

Depth, node, edge, and frontier limits are hard and return a typed limit error
(`BW8102`), so a malformed or adversarial graph cannot exhaust memory.
