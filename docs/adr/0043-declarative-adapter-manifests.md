# ADR 0043: Declarative, deterministic adapter manifests

- Status: Accepted
- Date: 2026-07-29

## Context

Adapters must be discoverable and comparable without executing arbitrary code.

## Decision

- Each adapter has a versioned manifest with a closed capability vocabulary and a deterministic content digest that excludes host-specific data.
- There is no mutable global registry; the built-in manifest set is immutable and callers receive copies.
- Unknown capabilities are rejected.

## Consequences

Adapter metadata is auditable, deterministic, and safe to load.
