# ADR 0004: Public contract boundaries

- Status: Accepted
- Date: 2026-07-29

## Context

The foundational domain layer introduces contracts that every later component
depends on. Their placement determines what is a stable public API versus an
implementation detail that can change freely.

## Decision

- The root `batchweaver` package holds the user-facing generic batch types
  (`BatchRequest`, `BatchResponse`, `Outcome`), the typed function contracts, and
  the typed declarations. It imports only `operation` and the standard library.
- `operation` holds the semantic domain model (IDs, symbols, semantics,
  contracts, policies, specs, catalog). It imports only `diagnostics` and the
  standard library.
- `config` holds the strict configuration schema and loader. It imports
  `operation` and `diagnostics` and the internal configuration packages; it never
  exposes internal decoder or node types.
- `diagnostics` holds the dependency-free diagnostic model. It imports nothing
  from BatchWeaver.
- All unstable implementation (decoding, discovery, merge, CLI) lives under
  `internal/`.

The one-way dependency direction is enforced by an architecture test
(`arch_test.go`) using standard-library parsing.

## Consequences

- The public surface is small, stable, and free of internal leakage.
- Later compiler and runtime code can import these contracts without cycles.
- Moving a package out of `internal/` remains a deliberate, reviewed decision.

## Alternatives considered

- **A single package for all contracts.** Rejected; it would couple diagnostics,
  the domain model, and configuration and make the dependency direction implicit.
- **Exposing the decoder node types publicly.** Rejected; they are an unstable
  implementation detail.
