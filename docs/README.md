# BatchWeaver Documentation

This directory contains the project's design and development documentation.

## Architecture

- [Overview](architecture/overview.md) — the end-to-end pipeline and its
  dependency direction.
- [Package boundaries](architecture/package-boundaries.md) — public versus
  internal packages and the rules that keep them separated.
- [Foundational domain model](architecture/foundational-domain-model.md) — the
  operation spec, contracts, and typed declarations.
- [Configuration pipeline](architecture/configuration-pipeline.md) — discovery,
  decoding, includes, merge, normalization, and digest.
- [Diagnostics pipeline](architecture/diagnostics-pipeline.md) — codes,
  positions, collection, sorting, and rendering.
- [Future compiler pipeline](architecture/future-compiler-pipeline.md) — the
  planned compiler stages (not yet implemented).

## Concepts

- [Batching model](concepts/batching-model.md) — the vocabulary BatchWeaver uses
  to talk about scalar and batch execution.
- [Operation contracts](concepts/operation-contracts.md)
- [Result semantics](concepts/result-semantics.md)
- [Isolation and partitioning](concepts/isolation-and-partitioning.md)
- [Scheduling policy](concepts/scheduling-policy.md)

## Reference

- [Configuration](reference/configuration.md) — every configuration field.
- [Typed declarations](reference/typed-declarations.md) — the declaration API.
- [Diagnostic codes](reference/diagnostic-codes.md) — the code registry.

## Development

- [Getting started](development/getting-started.md) — build, run, and test.
- [Quality gates](development/quality-gates.md) — the checks every change must
  pass.
- [Repository conventions](development/repository-conventions.md) — layout,
  naming, and commit conventions.

## Architecture Decision Records

- [ADR index](adr/README.md) — the log of significant decisions.
