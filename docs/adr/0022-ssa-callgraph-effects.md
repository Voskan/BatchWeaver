# ADR 0022: SSA, call graph, and effect strategy

- Status: Accepted
- Date: 2026-07-29

## Context

Call-site discovery and effect reasoning need SSA and a call graph. The strategy
must be conservative and deterministic, and must not overclaim dynamic-dispatch
resolution.

## Decision

- Build SSA with `go/ssa` (generic instantiation enabled), degrading gracefully
  when a package has type errors.
- Use the conservative CHA call graph (`go/callgraph/cha`), suitable for library
  code without program roots; direct calls stay exact and interface dispatch
  retains ambiguity.
- Match scalar-operation calls precisely for direct static calls, and
  conservatively (by method name and identical signature) for interface
  dispatch, marking them ambiguous.
- Compute conservative effect summaries with a bounded, monotone interprocedural
  fixed point; unresolved dynamic calls mark a summary incomplete, and unknown
  never collapses into optimistic absence.

## Alternatives considered

- RTA/VTA now: deferred; CHA is sufficient and explainable for this stage, and
  algorithms should not be added for checkbox coverage.
- Optimistic purity by naming: rejected; only reviewed standard-library models
  contribute effects.

## Consequences

Deterministic, conservative call and effect data that later proofs can trust.

## Compatibility

Analysis results may vary across Go toolchains; the Go version is recorded in the
snapshot.
