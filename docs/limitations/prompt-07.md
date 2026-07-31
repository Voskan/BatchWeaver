# Runtime lowering limitations (this stage)

This stage lowers certified scalar calls into a typed runtime bridge and
integrates with the Go build. This document records what it does not yet do.

## Supported

- Lowering `recv.Scalar(ctx, key)` for a certified read-only operation into a
  typed `bridge.Operation.Call`, for standalone, loop, straight-line sibling,
  and existing goroutine/errgroup fan-out call sites.
- A typed, reflection-free bridge with exact scalar fallback and a real runtime
  coalescing path when the application installs a bound operation.
- Overlay-first build/test/run, a `-toolexec` driver with recursion prevention,
  `tool-exec doctor`/`explain`, `runtime inspect`, and `barrier inspect`.
- Deterministic plans, generated bridge files, materialization/revert of both
  modified and generated files, and type-check validation through the overlay.

## Not supported yet

- Automatic backend batch-provider synthesis (SQL, Redis, gRPC, HTTP, etc.). Every
  lowered operation still requires a declared, compatible batch provider.
- Aggressive parent-level static enqueue for fan-out, and errgroup
  concurrency-limit (`SetLimit`) aware parent coalescing. The default lowers the
  call inside each goroutine/closure and preserves the concurrency envelope.
- Automatic installation of bound operations into scopes; the application wires
  `bridge.WithOperation` and declares the provider. Without it, lowered calls fall
  back to the scalar call.
- Automatic barrier insertion into source (the explicit `bridge.Flush` API and
  static barrier reporting are provided; conservative auto-insertion is staged).
- Runtime verification shadow execution wired into generated code (the model,
  read-only restriction, and policies are decided in ADR 0040; generated shadow
  calls are staged).
- Coverage remapping to original source beyond source-map identification, and
  full cross-platform `-toolexec` specifics beyond shell-free argument arrays.
- Lowering of call forms beyond `recv.Scalar(ctx, key)` (for example
  `v, found, err` three-result reads and `if v, err := call(); ...` init forms)
  is conservatively rejected until explicitly implemented.

## Non-guarantees

Runtime lowering preserves scalar semantics on the fallback path exactly. The
coalescing path depends on the declared operation contract and the correctness of
the application-installed bound operation for the scope's receiver. Correctness
assumes the analyzed program is free of data races, as certified by the proof.
This stage does not generate backend batch APIs and claims no performance
improvement without measured evidence.
