# Changelog

All notable changes to this project are documented in this file.

The format is based on [Keep a Changelog](https://keepachangelog.com/en/1.1.0/),
and this project adheres to [Semantic Versioning](https://semver.org/spec/v2.0.0.html).
The public API may evolve before the 1.0 release.

## [Unreleased]

### Added

- Explicit, typed request-coalescing runtime (`runtime`, imported as
  `batchruntime`): an instance-scoped `Engine`; typed operation `Bind` and
  `BoundOperation.Do`; explicit `Scope`s carried through context with flush,
  drain, and idempotent close; a typed `Provider` contract; key strategies
  (comparable, string, bytes) with collision-safe deduplication; opaque,
  privacy-preserving `Partition`s; bounded queues with reject, fallback, and
  block overflow policies; immediate, fixed-window, deadline-aware, and manual
  scheduling over a testable `Clock`; in-flight deduplication and opt-in
  scope-local memoization; independent per-caller cancellation and a dedicated
  provider deadline algorithm; provider-response validation; scalar fallback;
  recursion detection; backend-neutral event hooks; and immutable statistics
  snapshots. Concurrency is race-detector clean.
- Foundational operation domain model: strict operation IDs, Go symbol
  references, semantic kinds, and result, partition, scheduler, deduplication,
  retry, and fallback contracts, with cross-field validation that reports
  diagnostic collections. Operation catalogs reject duplicate IDs and produce a
  deterministic digest.
- Canonical generic batch contracts in the root package: `RequestID`,
  `BatchItem`, `BatchRequest`, `Outcome`, `BatchResponse`, typed scalar/batch
  function and method types, response-mapping helpers, and typed function and
  method declarations without global registration.
- Strict, versioned (schema 1) configuration system: YAML and JSON loading,
  local includes with cycle detection, deterministic merge, centralized
  defaults, normalization into an operation catalog, semantic validation,
  canonical JSON rendering, and a semantic digest that is identical for
  equivalent YAML and JSON.
- Expanded diagnostics: stable `BW<CATEGORY><NNN>` codes, severities, positions
  and ranges, related information, advisory fixes, a deterministic collection,
  and text and JSON formatters with golden tests.
- CLI commands `config validate`, `config print`, and `operation list`, with
  documented, stable exit codes.
- Dependency `github.com/goccy/go-yaml` (v1.19.2, MIT) for YAML parsing.
- Documentation: ADRs 0004–0009, architecture and concept documents, and a
  configuration, typed-declaration, and diagnostic-code reference.
- Compile-tested declaration and configuration examples.
- Initial repository foundation: scalable directory layout and package
  boundaries.
- Standard-library command-line interface with `version` and `help` commands.
- Foundational packages: `internal/buildinfo`, `internal/cli`,
  `internal/filesystem`, `internal/project`, `config`, `diagnostics`, and a
  documented `runtime` package.
- Open-source governance documentation and issue/pull-request templates.
- Architecture documentation and initial architecture decision records.
- Quality automation: formatting, vet, linting, vulnerability scanning,
  documentation and YAML linting, and continuous integration workflows.
