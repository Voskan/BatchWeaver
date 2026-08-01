# ADR 0045: PostgreSQL-first SQL dialect

- Status: Accepted
- Date: 2026-07-29

## Context

An initial dialect is needed for concrete generation.

## Decision

- The first synthesized dialect is PostgreSQL, using unnest($1::TYPE[]) WITH ORDINALITY, a LEFT JOIN on the key, and ORDER BY the request ordinal.
- Keys are bound as one typed array parameter; values are never interpolated.
- Other dialects are out of scope until added with their own tests.

## Consequences

Generated SQL is parameterized, ordered, and preserves missing/duplicate semantics for PostgreSQL.
