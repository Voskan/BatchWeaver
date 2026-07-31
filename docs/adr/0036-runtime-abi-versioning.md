# ADR 0036: Runtime ABI versioning

- Status: Accepted
- Date: 2026-07-29

## Context

Generated bridges and the runtime must agree on a stable interface that can evolve safely.

## Decision

- A versioned bridge ABI (batchweaver.bridge/v1alpha1) is recorded on every runtime transformation and in generated files.
- An ABI change invalidates generated bridges and plans through the plan digest.
- Unstable runtime internals are never exposed to generated code; only the small typed bridge surface is.

## Consequences

Generated code and runtime stay compatible, and stale bridges are detected and regenerated.
