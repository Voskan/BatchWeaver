# Changelog

All notable changes to this project are documented in this file.

The format is based on [Keep a Changelog](https://keepachangelog.com/en/1.1.0/),
and this project adheres to [Semantic Versioning](https://semver.org/spec/v2.0.0.html).
The public API may evolve before the 1.0 release.

## [Unreleased]

No changes yet.

## [0.1.0-beta.2] - 2026-08-02

### Fixed

- Derive CLI version and source revision from Go module build information when
  link-time release metadata is absent, so `go install ...@v0.1.0-beta.2`
  reports the installed beta instead of `dev`.
- Correct the GitHub Pages checkout action pin and deploy the documentation site
  through the protected workflow.
- Make prerelease publication helpers derive their immutable tag, confirmation,
  gate report, title, and notes from `release/VERSION`.

## [0.1.0-beta.1] - 2026-08-02

### Added

- Stable-release evidence triage and package-readiness documentation: a
  compile-tested pkg.go.dev example, module-consumption guide, current README,
  current architecture and package-boundary diagrams, reorganized documentation
  index, public API inventory, migration plan, blocked v1 decision, beta evidence,
  compatibility/security/performance/reproducibility/documentation reports, and
  post-v1 release and incident procedures. Stable v1 remains blocked because
  the beta evidence period, migration evidence, API freeze, and stable approval
  are incomplete.

### Fixed

- Enforce LF checkout for `.txt` fixtures so Windows compares generated output
  against golden and public-API baselines deterministically.
- Replace stale “future implementation” claims in package and architecture
  documentation with the implemented analyzer, compiler bridge, runtime, and
  transformation behavior.

- Public-beta preparation for `v0.1.0-beta.1`: machine-readable launch gates,
  authenticated publication stop conditions and idempotent helper scripts,
  deterministic GitHub Pages source/workflow, privacy-safe `doctor --bundle`,
  beta issue forms and label policy, support/security/governance and incident
  procedures, adoption/migration/rollout guides, launch-health and beta-exit
  criteria, and an unposted announcement/demo kit. Authenticated GitHub,
  Dependency Review, protected-main, security-reporting, community, and editor
  host gates are verified for the beta.

- Release-candidate assurance commands: `release check|build|verify|reproduce|
  notes|manifest|clean`, `compatibility report`, `verify differential`, and
  `security audit`; deterministic five-target binary archives and source archive;
  SHA-256 checksums; SPDX 2.3 and CycloneDX 1.5 SBOMs; unsigned local provenance;
  strict artifact verification and byte-reproduction; exported-Go-API baseline;
  schema compatibility and manifest fuzzing; central deterministic differential,
  modeled safety-mutation, fault-injection, short-soak, and performance-budget
  suites; explicit compatibility/security/performance/reproducibility reports;
  release policy, checklist, rollback, draft notes, and beta-launch handoff.
- A locked VS Code extension dependency graph, pinned VSIX packager, manifest
  consistency test, and removal of two unimplemented editor settings. Generated
  Mach-O binaries are no longer tracked in source control.

- Editor and interactive developer-experience layer: a standalone `batchweaver
  lsp` language server (LSP 3.17 over a small internal JSON-RPC implementation
  with hand-written protocol types and no gopls internal imports), an optional
  `--proxy-gopls` mode that launches and composes the user's gopls (forwarding
  standard Go traffic with automatic request-ID namespacing, merging initialize
  capabilities and hover/code-action/code-lens results, and keeping diagnostics
  separated by source), and a local workspace daemon (`batchweaver daemon
  start|status|stop|clean`) with a versioned Unix-socket protocol, discovery,
  health, and lifecycle. Open editor buffers are authoritative: BatchWeaver
  analyzes unsaved bytes through a `go/packages` overlay and publishes live,
  debounced, snapshot-consistent diagnostics (including the `BW1001` batching
  opportunity), hover, code lenses, and preview code actions, using a single
  canonical UTF-16/byte position mapper. Adds `batchweaver editor doctor`, a VS
  Code extension source tree (sidecar/proxy modes, commands, settings, status
  bar, output channel, and workspace-trust handling), editor setup guides for
  Neovim, Emacs/Eglot, Helix, and Zed, an editor support matrix, and LSP/editor
  diagnostics in the `BW9xxx` range. No source is ever written implicitly and no
  remote telemetry is collected. ADRs 0072–0081. No new Go module dependencies.
- Adaptive scheduling and production tuning layer (`internal/adaptive`) and the
  `batchweaver profile`, `tune`, `fairness`, `overload`, `wave`, and `recursive`
  commands. Privacy-safe, versioned workload profiles
  (`batchweaver.profile/v1alpha1`) store only bounded, mergeable histograms and
  anonymized categorical counts — never raw keys, tenants, tokens, payloads, or
  parameters — with atomic checksummed persistence, compatibility and staleness
  checks, and merge. A versioned cost model (`batchweaver.cost/v1alpha1`) with
  explicit objective weights drives a bounded, explainable controller that
  recommends `max_wait`, `max_batch_size`, concurrency, chunk size, and execution
  mode, with cold/warm start, shadow and active modes, canary and exploration
  limits, phase-change detection, SLO guardrails, and automatic rollback; every
  decision is content-addressed with evidence, reasons, and confidence. The
  runtime applies accepted settings atomically through
  `BoundOperation.ApplyAdaptiveSettings`, clamped to the binding's hard
  configuration limits. A multi-operation wave DAG (`batchweaver.wave/v1alpha1`)
  co-schedules independent operations and computes waves and the critical path;
  recursive breadth-first batching loads proven traversals one batched call per
  frontier with explicit cycle, ordering, error, and limit policies. Fairness
  (weighted fair queueing and deficit round robin) adds priority classes,
  quotas, reserved capacity, and starvation detection; overload control adds
  admission policies, non-silent load shedding, and backpressure. Deterministic
  offline replay, policy simulation, counterfactuals, seeded synthetic workload
  generators, and text/JSON/Markdown tuning reports round out the layer.
  Diagnostics use the `BW8xxx` range, kept distinct from earlier stages. No new
  module dependencies.
- Network protocol adapters extending the adapter SDK, and the `batchweaver
  graphql`, `grpc`, `http`, and `openapi` commands (plus `adapter list
  --category`). A fully implemented HTTP explicit-batch adapter over `net/http`
  maps typed keyed/positional JSON envelopes to ordered outcomes with correlation
  validation, deterministic chunking, and preserved transport/auth identity,
  verified hermetically with `httptest`; OpenAPI 3.1+ documents (JSON via the
  standard library, YAML via the existing goccy dependency) are loaded with a
  bounded size and no remote references, discovering `x-batchweaver` batch
  bindings. A framework-neutral GraphQL model with a recursive-descent query
  parser (no regex, never panics) computes resolver execution waves, normalized
  selection digests, and one scope per operation, preserving field paths,
  errors, nullability, and authorization/selection partitions. An explicit gRPC
  batch-binding model with strict validation, metadata partition policy
  (authorization/tenant/routing never merged across callers), response-correlation
  modes, and a streaming lifecycle vocabulary is provided. Protocol contract
  verification reuses the adapter harness. Network diagnostics use the `BW7xxx`
  range (BW71xx GraphQL, BW72xx gRPC, BW73xx HTTP/OpenAPI). Concrete gqlgen and
  grpc-go client integrations are contract-defined but deferred in this build
  (offline dependency); no new module dependencies were added. No universal
  GraphQL optimization, remote batch-method generation, or arbitrary HTTP request
  fusion. See docs/limitations/network-adapters.md. ADRs 0052-0061.
- Backend adapter SDK (`internal/adapter`) and the `batchweaver adapter`
  (`list`, `inspect`, `explain`, `verify`, `doctor`) command. It provides a
  versioned, deterministic manifest and closed capability model with no mutable
  global registry; a narrow, real SQL parser (a hand-written tokenizer and strict
  recursive parser, never regex-primary, never panics) that safely synthesizes
  exact-key PostgreSQL read batches (`unnest($1::TYPE[]) WITH ORDINALITY` +
  `LEFT JOIN` + `ORDER BY` ordinal, fully parameterized) and rejects every
  unsupported construct with an exact code, node, and byte offset; a typed,
  reflection-free `database/sql` runtime batch provider that maps rows by request
  ordinal, preserving order, duplicates, and `sql.ErrNoRows`, with deterministic
  chunking; Redis cluster hash-slot computation (CRC-16/XMODEM with hash tags) and
  slot grouping plus MGET/HMGET/pipeline mapping logic; and a scalar/batch
  contract-verification harness that emits a deterministic contract artifact.
  Adapter diagnostics use the `BW6xxx` range (distinct from the proof stage's
  `BW5xxx`). The concrete pgx v5 and go-redis v9 client bindings are
  contract-defined but deferred in this build (offline dependency); no arbitrary
  SQL transformation or automatic write synthesis. See
  docs/limitations/backend-adapters.md. ADRs 0042-0051.
- Runtime call lowering and the public `bridge` package (ABI
  `batchweaver.bridge/v1alpha1`). Certified scalar calls are rewritten to a typed,
  reflection-free `bridge.Operation.Call` that routes through the typed runtime
  when a scope with an installed bound operation is active and otherwise calls the
  scalar function directly (exact fallback). The same lowering covers standalone,
  loop, straight-line sibling, and existing goroutine/errgroup fan-out call sites
  without introducing new concurrency, preserving context, cancellation cause,
  deadlines, receiver identity, partitions, and error semantics. Adds strategy
  IDs `runtime-call-coalescing`, `static-sibling-fusion`, `fanout-coalescing`, and
  `errgroup-coalescing`; deterministic generated bridge files
  (`zz_batchweaver_<op>_gen.go`) with the standard generated-code header; created
  (generated) files in plans, overlays, materialization, and revert; an
  overlay-first `-toolexec` driver (`batchweaver toolexec`) with recursion
  prevention and environment hygiene, plus `tool-exec doctor`/`explain`; explicit
  `bridge.Flush`/`Barrier` batching barriers and `barrier inspect`; and
  `runtime inspect`. `transform plan`, `build`, `test`, and `run` gain a
  `--strategy` selector, and build/test/run separate BatchWeaver flags from Go
  arguments with `--`. Every lowered operation still requires a declared batch
  provider; no backend batch synthesis. See docs/limitations/runtime-lowering.md.
- Transformation engine (`internal/transform`, `internal/gocommand`) and the
  `batchweaver transform` (plan, diff, inspect, verify, materialize, revert,
  clean, recover) and `build`, `test`, `run` commands. It consumes semantic proof
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
  only; see docs/limitations/transformation.md.
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
