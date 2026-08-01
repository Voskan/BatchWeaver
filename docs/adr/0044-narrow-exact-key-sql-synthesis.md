# ADR 0044: Narrow exact-key SQL synthesis

- Status: Accepted
- Date: 2026-07-29

## Context

Automatic SQL batching is only safe for a tiny, well-understood query shape.

## Decision

- Only exact-key SELECTs (single equality on a key parameter, explicit projection, single relation, optional key-independent IS [NOT] NULL predicates) are synthesized.
- A hand-written tokenizer plus a strict recursive parser accepts that grammar and rejects everything else with an exact code, node, and offset.
- Every other construct (joins, GROUP BY, ORDER BY, LIMIT, locking, functions, set ops, writes, multiple statements) is rejected.

## Consequences

Automatic synthesis is conservative and never guesses SQL semantics.
