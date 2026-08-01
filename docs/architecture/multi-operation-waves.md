# Multi-operation waves

The wave planner (`batchweaver.wave/v1alpha1`) coordinates work across
operations. It models an operation dependency DAG and dispatches it in waves.

## Nodes and edges

Nodes are operation executions, pure computations, barriers, adapter-compound
requests, or recursive frontiers. Edges express data, control, barrier,
partition, transaction/session, and error-order dependencies.

## Waves

Independent nodes at the same dependency level form one wave: they are
co-scheduled (run in parallel, coordinated wait) but not merged into a single
batch unless they share a declared fusion group. Waves and the critical path are
computed deterministically with a topological longest-path algorithm; a cycle is
reported as `BW8101` and is a hard error.

## Provider fusion

Where a Prompt 08/09 adapter explicitly declares a compound capability (one
GraphQL document with multiple fields, one HTTP batch envelope, one gRPC
multiplexed request), nodes sharing a fusion group can be issued as one compound
request. BatchWeaver provides the scheduling hook only; it never invents an
unsupported compound protocol.
