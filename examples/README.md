# Examples

Runnable, compile-tested examples of BatchWeaver's foundational contracts. These
demonstrate the data and type contracts only — no automatic batching, scheduling,
or transformation is performed yet.

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
