# ADR 0024: Conservative unknown, no optimistic defaults

- Status: Accepted
- Date: 2026-07-29

## Context

A false positive in the proof engine can cause a later stage to rewrite correct
Go into behaviorally different code. Absence of evidence is not evidence of
safety.

## Decision

- Every obligation is evaluated over a lattice
  (`satisfied`, `violated`, `needs_assumption`, `unknown`, `not_applicable`,
  `deferred`), never a boolean.
- Precision loss, incomplete effect summaries, unresolved interface targets, and
  resource-limit exhaustion all yield `unknown`, not eligibility.
- An operation is never treated as pure because of its name; only a validated
  contract or a modeled standard-library fact establishes read-only behavior.
- Unresolved interface targets are never merged into one optimistic target.

## Consequences

The engine frequently reports `unknown`. This is a correct and necessary result:
`unknown` blocks transformation without falsely rejecting a candidate that a
future, more precise analysis could prove.
