# Changelog

All notable changes to this project are documented in this file.

The format is based on [Keep a Changelog](https://keepachangelog.com/en/1.1.0/),
and this project adheres to [Semantic Versioning](https://semver.org/spec/v2.0.0.html).
The public API may evolve before the 1.0 release.

## [Unreleased]

### Added

- Transformation engine (`internal/transform`, `internal/gocommand`) and the
  `batchweaver transform` (plan, diff, inspect, verify, materialize, revert,
  clean, recover) and `build`, `test`, `run` commands. It consumes Prompt 05 proof
  certificates and performs the first production transformation — static
  slice/array loop prefetch for a certified read-only operation with an ordered,
  global-error batch provider: typed key collection in source order, a single
  batch call, global-result validation mirroring the scalar error return, and
  source-order replay. It produces a deterministic, schema-versioned
  (`batchweaver.transform/v1alpha1`) plan with content-addressed IDs, a unified
  diff, and a source map; validates every plan by parsing and type-checking the
  affected packages through an overlay; and executes transformed code through a Go
  `-overlay` without modifying source. Materialization is explicit, atomic, and
  reversible with a backup manifest, conflict-aware revert, and idempotent
  recovery. Added an end-to-end `examples/static-prefetch` package with a semantic
  equivalence harness. Supports a certified subset of read-only slice/array loops
  only; see docs/limitations/prompt-06.md.
- Semantic proof engine (`internal/proof`) and the `batchweaver prove`,
  `candidate inspect`, `proof inspect|explain|graph`, `assumption list`, and
  `strategy list|inspect` commands. The engine consumes the analysis snapshot and
  validated operation contracts and decides, for every discovered candidate, a
  strategy-specific eligibility outcome — proven eligible, proven ineligible,
  requires assumption, unknown, or deferred — from a closed registry of
  Go-semantic and operation-contract proof obligations evaluated on a status
  lattice. It reasons about evaluation order and effect barriers, key dependency
  (structural, result-dependent, call-derived), receiver and context invariance,
  partition stability, result and first-error reconstruction, defer/recover
  boundaries, and concurrency envelopes; records scoped assumptions and a
  data-race-free trust boundary; and emits deterministic, evidence-backed proof
  certificates with witnesses, `BW5xxx` diagnostics, and a schema-versioned
  (`batchweaver.proof/v1alpha1`) text/JSON report with a `--reproducible` mode.
  No source is modified and no execution is changed.
- Analysis refinements consumed by the proof engine: builtins (`append`, `len`,
  `make`, …) are classified as benign rather than unknown calls; each call site
  records its enclosing-function effect-summary ID and a conservative key
  dependency classification.
- Static analysis foundation (`internal/analysis`) and the `batchweaver scan`
  command: real Go module and package loading via `golang.org/x/tools/go/packages`
  under an explicit build context; canonical, portable identities and paths;
  operation discovery from typed declarations (by AST inspection) and
  configuration, merged with provenance and conflict diagnostics; symbol
  resolution and conservative signature compatibility; SSA construction
  (`go/ssa`); a conservative CHA call graph; scalar-operation call-site indexing
  with loop and goroutine-fan-out context; conservative interprocedural effect
  summaries; a candidate inventory with explicit states; a deterministic,
  schema-versioned (`batchweaver.analysis/v1alpha1`) snapshot with text and JSON
  output and a `--reproducible` mode; and an `operation inspect` command. No
  source is modified and no batching safety is claimed.
- Dependency `golang.org/x/tools` (v0.48.0, BSD-3-Clause) for package loading,
  SSA, and call-graph analysis.
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
