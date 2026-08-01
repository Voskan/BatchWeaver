# Breadth-first batching

Breadth-first batching turns a recursive traversal that would issue one backend
call per node into one batched call per frontier level.

It applies only to traversals with an explicit contract and a valid semantic
proof. Breadth-first order and source child order are preserved — BatchWeaver
never substitutes DFS for BFS unless the declared semantics permit it — and
cycle, ordering, and error policies are explicit. Hard depth, node, edge, and
frontier limits bound the traversal and return typed errors.

See the [recursive batching architecture](../architecture/recursive-batching.md).
