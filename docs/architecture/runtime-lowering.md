# Runtime call lowering

Runtime lowering rewrites a certified scalar call site into a typed bridge call
that routes through the Prompt 03 runtime when a scope is active and otherwise
calls the original scalar function directly. It builds on the Prompt 06
transformation IR, overlays, and materialization rather than a second engine.

## What is lowered

A certified call `recv.Scalar(ctx, key)` becomes `bwopX.Call(ctx, recv, key)`,
where `bwopX` is a package-level bridge generated into a
`zz_batchweaver_<op>_gen.go` file in the same package. The same rewrite is used
for standalone calls, loop bodies, straight-line siblings, and calls already
running inside goroutine or errgroup fan-out — a single auditable primitive
(see [ADR 0038](../adr/0038-bridge-lowering-default-fanout.md)).

## Gating

A call site is lowered only when a current proof certificate proves its candidate
eligible for a runtime strategy (`runtime-scope-coalescing` for sequential
call sites, `existing-fanout-coalescing` for concurrent ones) and the user
requested a runtime strategy. Static loop prefetch takes precedence when both are
requested and applicable. The transformer never silently switches between static
prefetch and runtime lowering.

## Strategies

- `runtime-call-coalescing` — standalone, loop, or sequential call sites.
- `static-sibling-fusion` — straight-line sibling calls (lowered through the
  bridge in lexical order).
- `fanout-coalescing` / `errgroup-coalescing` — calls already running
  concurrently; the bridge coalesces naturally overlapping calls without adding
  concurrency.

## Safety

Every plan parses and type-checks the transformed and generated files through an
overlay before it is buildable. The bridge preserves scalar semantics exactly on
the fallback path and defers coalescing correctness to the certified contract and
the runtime. See [the generated bridge ABI](generated-bridge-abi.md) and
[batching barriers](../concepts/batching-barriers.md).
