# Architecture Decision Records

This directory records significant architectural decisions and the reasoning
behind them, so that future contributors can understand why the project is
shaped the way it is.

## Template

```markdown
# ADR NNNN: Title

- Status: Accepted
- Date: YYYY-MM-DD

## Context

## Decision

## Consequences

## Alternatives considered
```

## Index

- [ADR 0001 — Project architecture](0001-project-architecture.md)
- [ADR 0002 — Go version policy](0002-go-version-policy.md)
- [ADR 0003 — Agent state is local only](0003-agent-state-is-local-only.md)
- [ADR 0004 — Public contract boundaries](0004-public-contract-boundaries.md)
- [ADR 0005 — Canonical batch request and response](0005-canonical-batch-request-response.md)
- [ADR 0006 — Strict versioned configuration](0006-strict-versioned-configuration.md)
- [ADR 0007 — Stable diagnostic codes](0007-stable-diagnostic-codes.md)
- [ADR 0008 — Declarations without global registration](0008-declarations-without-global-registration.md)
- [ADR 0009 — YAML decoder selection](0009-yaml-decoder-selection.md)
- [ADR 0010 — Instance-scoped runtime engine](0010-instance-scoped-engine.md)
- [ADR 0011 — Explicit runtime scope model](0011-explicit-scope-model.md)
- [ADR 0012 — Key strategy and collision-safe deduplication](0012-key-strategy-and-collision-safety.md)
- [ADR 0013 — Opaque partition representation and privacy](0013-opaque-partition-representation.md)
- [ADR 0014 — Provider context and deadline algorithm](0014-provider-context-and-deadline.md)
- [ADR 0015 — In-flight deduplication versus scope memoization](0015-dedup-versus-memoization.md)
- [ADR 0016 — Deterministic scheduler and clock abstraction](0016-scheduler-and-clock.md)
- [ADR 0017 — Queue overflow policies](0017-queue-overflow-policies.md)
- [ADR 0018 — Provider and callback panic isolation](0018-provider-panic-isolation.md)
- [ADR 0019 — Runtime event-hook boundary](0019-event-hook-boundary.md)
- [ADR 0020 — Static analysis architecture and identities](0020-static-analysis-architecture.md)
- [ADR 0021 — Declaration discovery and precedence](0021-declaration-discovery-precedence.md)
- [ADR 0022 — SSA, call graph, and effect strategy](0022-ssa-callgraph-effects.md)
- [ADR 0023 — Proof as strategy-specific obligations](0023-proof-as-strategy-obligations.md)
- [ADR 0024 — Conservative unknown, no optimistic defaults](0024-conservative-unknown-no-optimism.md)
- [ADR 0025 — Explicit assumptions and the data-race-free prerequisite](0025-assumptions-and-data-race-free.md)
- [ADR 0026 — Deterministic proof identity and invalidation](0026-deterministic-proof-identity.md)
- [ADR 0027 — Proof certificates are separate from transformation](0027-certificates-separate-from-transformation.md)
- [ADR 0028 — Versioned transformation IR independent of AST/SSA](0028-versioned-transformation-ir.md)
- [ADR 0029 — Proof-certificate-gated rewrites](0029-certificate-gated-rewrites.md)
- [ADR 0030 — Go command overlays as the default non-mutating path](0030-overlays-default-non-mutating.md)
- [ADR 0031 — Explicit, atomic materialization with backup and revert](0031-atomic-materialization-and-revert.md)
- [ADR 0032 — Static slice/array loop prefetch as the first strategy](0032-static-loop-prefetch-first-strategy.md)
- [ADR 0033 — No transformation without type-check validation; deterministic names](0033-typecheck-and-deterministic-names.md)

## Process

Propose a new ADR as part of the pull request that introduces the decision. Give
it the next sequential number, set its status, and link it from this index.
Superseded ADRs are kept for history and marked as superseded rather than
deleted.
