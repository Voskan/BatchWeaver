# ADR 0025: Explicit assumptions and the data-race-free prerequisite

- Status: Accepted
- Date: 2026-07-29

## Context

Some facts cannot be inferred from Go semantics alone: the purity of a custom
wrapper, the identity of a transaction carried through an interface, or the
race-freedom of concurrent code. The proof model must accept such facts only
explicitly and record their influence.

## Decision

- Facts that cannot be inferred are represented as scoped assumptions with a
  stable ID, symbol scope, declared facts, digest, and origin.
- An assumption satisfies only the obligation types that reference the fact it
  declares; it never satisfies unrelated obligations and never overrides a hard
  Go-language fact.
- Assumptions are never applied automatically, except the built-in
  data-race-free prerequisite (`BW-A-RACEFREE`), which is recorded in every
  certificate whose eligibility depends on shared-memory reasoning.
- Reports distinguish inferred evidence from assumed evidence.

## Consequences

Guarantees are only as strong as their recorded assumptions, and those
assumptions are visible in every certificate. Definitive race detection remains
the responsibility of the Go race detector.
