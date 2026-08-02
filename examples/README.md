# Examples

Runnable, compile-tested examples of BatchWeaver's public contracts, runtime,
proof-gated transformations, adaptive controls, and editor integration. Each
example states whether it executes batching, previews a transformation, or only
demonstrates a declarative contract.

- [declarations](declarations) — declaring operations with typed scalar and batch
  contracts.
- [configuration](configuration) — valid YAML and JSON configuration, semantically
  equivalent.
- [static-prefetch](static-prefetch) — the static-loop-prefetch transformation,
  proven equivalent to the original N+1 loop.
- [adaptive-runtime](adaptive-runtime) — profiling a synthetic workload and
  getting a bounded, shadow-mode tuning recommendation.
- [multi-operation-wave](multi-operation-wave) — co-scheduling independent
  operations into dispatch waves and finding the critical path.
- [recursive-batching](recursive-batching) — loading a tree breadth-first, one
  batched call per frontier level.
- [fairness-overload](fairness-overload) — weighted fair scheduling across
  anonymized classes and overload admission decisions.
- [editor-diagnostics](editor-diagnostics) — an N+1 loop that the BatchWeaver
  language server flags live in an editor.
- [editor-transform-preview](editor-transform-preview) — the read-only
  transformation-preview flow in an editor.
- [editor-proxy](editor-proxy) — running BatchWeaver in proxy mode alongside
  gopls.

The `editor-*` fixtures are standalone Go modules; open them in an editor with
the BatchWeaver language server running.
