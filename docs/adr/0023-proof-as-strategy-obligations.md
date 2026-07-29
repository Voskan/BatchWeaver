# ADR 0023: Proof as strategy-specific obligations, not a safety score

- Status: Accepted
- Date: 2026-07-29

## Context

A batching transformation is safe for some shapes and unsafe for others. A single
boolean or numeric "safety score" hides which property holds and cannot tell a
later rewriting stage which transformation it may apply.

## Decision

- Model eligibility as a closed registry of named proof obligations
  (`BW-PROOF-<FAMILY>-<NNN>`) evaluated per candidate.
- Attach eligibility to a named strategy (for example `static-loop-prefetch`);
  never expose a generic `safe=true`.
- A candidate may be eligible for one strategy and unknown or ineligible for
  another; every strategy carries its own required-obligation set.
- The candidate decision aggregates strategy outcomes with a fixed precedence.

## Alternatives considered

- A single confidence score: rejected; it is not auditable and cannot drive a
  correct rewrite.
- One global obligation set for all strategies: rejected; runtime coalescing and
  static prefetch have materially different requirements.

## Consequences

Decisions are auditable and map directly to a strategy that a later stage may
implement. Obligation IDs are stable and never reused for a new meaning.
