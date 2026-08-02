# ADR 0044: Narrow exact-key SQL synthesis

- Status: Accepted
- Date: 2026-07-29

## Context

Automatic SQL batching is only safe for a tiny, well-understood query shape.

## Decision

- Exact/composite-key SELECTs use one or more contiguous parameter equalities,
  an explicit projection, and optional key-independent IS [NOT] NULL predicates.
- One qualified INNER/LEFT equality join is accepted only when synthesis is
  given an explicit at-most-one cardinality contract. Schema uniqueness is never
  inferred.
- A hand-written tokenizer plus a strict recursive parser accepts that grammar and rejects everything else with an exact code, node, and offset.
- Every other construct (unbounded/multiple joins, GROUP BY, ORDER BY, LIMIT,
  locking, functions, set ops, writes, multiple statements) is rejected.

## Consequences

Automatic synthesis is conservative and never guesses SQL semantics.
