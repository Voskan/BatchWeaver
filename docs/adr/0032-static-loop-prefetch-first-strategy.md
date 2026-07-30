# ADR 0032: Static slice/array loop prefetch as the first strategy

- Status: Accepted
- Date: 2026-07-29

## Context

The first production transformation should be the smallest end-to-end slice that
delivers real value with the strongest safety guarantees.

## Decision

- The first strategy is static loop prefetch for a certified, supported shape: a
  `for _, v := range sliceOrArray` loop whose first body statement is
  `value, err := receiver.Scalar(ctx, key)`, followed by an `if err != nil {
  return ... }` guard, over a read-only operation with an ordered, global-error
  batch provider `func(context.Context, []K) ([]V, error)`.
- The generator relies on the certificate and the concrete resolved types, not
  pattern matching alone; any deviation from the supported shape is a conservative
  skip with a reason.
- Map range, channel range, integer range, writes, conditional/early-exit body
  forms beyond the supported guard, deduplication, and canonicalization are out of
  scope for this stage.

## Consequences

A narrow but fully correct, type-checked, semantically verified transformation.
Broader forms are added in later stages.
