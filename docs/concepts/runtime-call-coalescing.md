# Runtime call coalescing

Runtime call coalescing lowers a certified scalar call into a typed bridge call
that routes through the Prompt 03 runtime. When several lowered calls run in the
same scope (sequentially through repeated invocation, or concurrently through
existing fan-out), the runtime coalesces compatible requests into batch provider
calls while preserving per-caller context, cancellation, deadlines, partitions,
and error identity.

## Behavior

- With an active scope and an installed bound operation, `Call` enqueues the
  request and the runtime batches compatible items.
- With no scope, `Call` invokes the scalar function directly (direct fallback),
  exactly preserving the original behavior.
- Deduplication, memoization, partitioning, and overflow follow the Prompt 03
  operation policy; the bridge never duplicates that logic.

## Execution modes

A lowered call may execute as runtime-coalesced, batch-of-one, or direct-scalar
fallback. Mode selection is the runtime's; the bridge only routes.

## Supported call forms

`v, err := recv.Scalar(ctx, key)` and the same call used in assignments, loop
bodies, sibling sequences, and inside existing goroutine or errgroup fan-out. The
scalar operation must be certified read-only with a declared batch provider.
