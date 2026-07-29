# ADR 0027: Proof certificates are separate from transformation

- Status: Accepted
- Date: 2026-07-29

## Context

The proof engine decides eligibility; a later stage performs source rewriting.
Coupling the two would make the proof engine harder to test and would risk
changing application behavior from within an analysis-only stage.

## Decision

- The proof engine produces certificates only. It does not rewrite source,
  intercept the build, insert runtime calls, or generate providers.
- Certificates record the exact mapping algorithm and obligations a future
  transformation must honor, so the rewrite stage consumes them without
  re-deriving candidate structure, contracts, or eligibility.
- The proof schema version is independent from the analysis schema version.

## Consequences

The proof engine is analysis-only and side-effect-free with respect to the
analyzed program. The first transformation stage (a later prompt) consumes these
certificates as its trusted input.
