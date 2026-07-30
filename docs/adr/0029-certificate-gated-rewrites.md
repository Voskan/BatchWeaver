# ADR 0029: Proof-certificate-gated rewrites

- Status: Accepted
- Date: 2026-07-29

## Context

A rewrite that is not backed by a current, valid semantic proof can change
program behavior. Analysis finding "no conflict" is not permission to rewrite.

## Decision

- A candidate is transformed only when a current proof certificate proves it
  eligible for the exact requested strategy (`proven_eligible` overall and for
  the strategy), with all required assumptions present and invalidation inputs
  matching current reality.
- The strategy consumes a typed, validated certificate object, never raw proof
  JSON, and never accepts a certificate by ID alone.
- A failed static transformation is never silently downgraded to another
  strategy; it is skipped with a reason.

## Consequences

Every rewrite is traceable to a specific certificate and obligation set. Stale,
unknown, deferred, ineligible, or assumption-missing candidates are refused.
