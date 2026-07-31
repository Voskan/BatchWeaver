# ADR 0040: Read-only shadow verification is opt-in and safe

- Status: Accepted
- Date: 2026-07-29

## Context

Runtime verification that compares transformed and scalar execution must never duplicate observable writes.

## Decision

- Verification is restricted to read-only operations and is off by default, with off/sample/always modes.
- It never shadows non-idempotent writes and requires an explicit read-only policy.
- The comparator model and mismatch policy are defined; wiring shadow execution into generated code is staged behind this decision.

## Consequences

Verification cannot cause unauthorized duplicate side effects.
