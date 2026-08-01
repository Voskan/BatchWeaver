# ADR 0050: Adapter contract verification artifacts

- Status: Accepted
- Date: 2026-07-29

## Context

Adapter correctness must be checkable by comparing scalar and batch behavior.

## Decision

- A versioned verification harness compares scalar and batch outcomes across cases (unique, duplicate, missing, empty, one, error) and emits a deterministic contract artifact with a digest.
- Values are compared with a caller comparator; errors are compared by nilness and errors.Is.
- Verification is read-only and never shadows writes.

## Consequences

Adapter bindings are verifiable and produce an auditable contract digest.
