# ADR 0034: Typed generated bridge instead of reflection

- Status: Accepted
- Date: 2026-07-29

## Context

The generated bridge must integrate with the typed Prompt 03 runtime without reflection on the call path.

## Decision

- Generated code declares a typed bridge.Operation[R, K, V] per operation and calls its Call method; there is no reflection on the call path.
- Call routes through an installed typed runtime bound operation when present and otherwise invokes the scalar function directly.
- Result-shape dispatch never uses runtime type assertions; the only assertion is a single O(1) context fetch of the typed bound operation.

## Consequences

Generated call sites are typed and allocation-light, and the fallback path is exactly the original scalar call.
