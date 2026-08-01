# ADR 0057: gRPC metadata and call-option partitioning

- Status: Accepted
- Date: 2026-07-30

## Context

Coalescing must not merge security or routing metadata across callers.

## Decision

- Metadata keys are classified (must-equal, partition, merge, forbidden); authorization, credentials, tenant, and routing keys partition so different callers never share a batch; tracing keys may merge; unknown keys partition conservatively.
- Call options that affect semantics partition or reject; unknown options are conservative.
- gRPC status codes, messages, and details are preserved per item and at the batch level.

## Consequences

No cross-caller credential or routing leakage; status semantics are preserved.
