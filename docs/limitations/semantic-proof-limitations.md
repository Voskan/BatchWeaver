# Semantic proof limitations

The proof engine is intentionally conservative. This document records what it
does and does not establish, so its results are never overinterpreted.

## What "proof" means here

A proof is a conservative static derivation under recorded assumptions, not a
theorem about arbitrary external systems or racy programs. A proven-eligible
certificate states that a named transformation strategy preserves the observable
scalar semantics for the candidate, given the recorded assumptions and the
declared operation contract.

## What is not guaranteed

Every certificate records that it does not guarantee provider performance,
provider side effects not declared in the operation contract, or the behavior of
code reached only through unresolved reflection. Certificates involving
concurrency additionally record the data-race-free assumption.

## Conservative boundaries

- **Coarse effect summaries.** Order and dependency obligations use whole-function
  effect summaries. An observable effect anywhere in the enclosing function makes
  static movement `unknown` rather than proving a specific between-call barrier.
- **Contracts required.** Result, error, and partition obligations depend on a
  validated operation contract; without one they are `unknown`.
- **Interface dispatch.** Unresolved interface targets are `unknown` and are never
  merged into one optimistic target.
- **Reflection, unsafe, cgo, assembly, plugins.** These produce unknown effects
  and block proofs unless a trusted model narrows them.
- **Map, channel, and range-over-function iteration.** Static prefetch over these
  is not proven by the core engine; runtime coalescing is evaluated separately.
- **Writes and aggregations.** The core engine does not batch writes or
  aggregations; such candidates are ineligible or deferred.
- **Resource limits.** Exceeding the configured candidate budget yields `unknown`
  with a resource-limit reason, never eligibility.

## Not implemented in this stage

The proof engine does not rewrite source, intercept the build, hoist loops,
insert runtime calls, generate providers, synthesize backend queries, or change
application execution. It does not claim that N+1 calls have been eliminated. It
produces proof certificates that a later transformation stage consumes.
