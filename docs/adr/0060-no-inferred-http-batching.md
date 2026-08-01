# ADR 0060: No inferred HTTP batching

- Status: Accepted
- Date: 2026-07-30

## Context

Plural or batch-looking endpoint names do not imply batch semantics.

## Decision

- An HTTP batch endpoint is used only when explicitly declared (extension or configuration) with a typed request envelope and a stable response correlation (keyed or strictly positional).
- A batch endpoint with no stable item correlation key is rejected with a diagnostic.
- Transport, TLS, cookie jar, redirect policy, and authentication identity partition calls; different auth identities never batch.

## Consequences

HTTP batching is never guessed; correlation and transport identity are preserved.
