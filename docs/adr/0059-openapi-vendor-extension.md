# ADR 0059: OpenAPI vendor-extension schema

- Status: Accepted
- Date: 2026-07-30

## Context

HTTP batch relationships must be declared, not inferred, and travel with the API document when possible.

## Decision

- A versioned x-batchweaver OpenAPI extension declares the scalar operation, envelope mode, item/key/error paths, and item limits; it is validated strictly.
- OpenAPI 3.1+ documents (JSON or YAML) are loaded with bounded size and no remote reference resolution by default.
- External user configuration may declare the same relationship when the document cannot be modified.

## Consequences

HTTP batch bindings are explicit, portable, and safe to load.
