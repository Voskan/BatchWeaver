// Package proof implements BatchWeaver's semantic batching proof engine.
//
// The proof engine consumes the deterministic analysis snapshot produced by the
// analysis package and decides, for every structurally discovered batching
// candidate, whether one or more precisely named transformation strategies are
// semantically eligible. It never rewrites source, never executes analyzed
// application code, and never equates "no issue detected" with proof of
// equivalence.
//
// # Trust boundary
//
// The engine reasons only from facts it can justify: the Go language and memory
// model, the standard-library synchronization semantics modeled in the analysis
// package, validated BatchWeaver operation contracts, and the conservative
// analysis facts (targets, effects, structural context). Everything else is
// unknown until an explicit, scoped assumption supplies it. Precision loss and
// resource exhaustion yield the unknown decision, never an optimistic one.
//
// # Decisions
//
// Each candidate receives exactly one of five decisions — proven eligible,
// proven ineligible, requires assumption, unknown, or deferred — derived from a
// closed registry of strategy-specific obligations rather than a single opaque
// safety score. A candidate may be eligible for one strategy and unknown or
// ineligible for another; eligibility is always reported per strategy.
//
// # Non-goals
//
// This package produces proof certificates only. It does not implement source
// rewriting, loop hoisting, runtime-call insertion, provider generation, or any
// build interception; those are reserved for later stages that consume the
// certificates emitted here.
package proof
