# ADR 0061: Protocol contract verification artifacts

- Status: Accepted
- Date: 2026-07-30

## Context

Protocol correctness must be checkable with hermetic fixtures.

## Decision

- The Prompt 08 verification harness is reused for protocols, comparing scalar and batch behavior (values, errors, correlation, missing/duplicate) over hermetic httptest/in-memory fixtures and emitting a deterministic contract artifact.
- Diagnostics for network adapters use the BW7xxx range (BW71xx GraphQL, BW72xx gRPC, BW73xx HTTP/OpenAPI), distinct from backend adapters (BW6xxx).
- Artifacts contain no credentials or raw payloads.

## Consequences

Protocol bindings are verifiable and auditable without network access.
