# BatchWeaver

BatchWeaver is a proof-gated Go compiler/runtime toolkit that turns supported
scalar access patterns into reviewed batch execution while preserving explicit
semantic and isolation contracts.

> **Public beta preparation — v0.1.0-beta.1.** The source repository is public,
> but this version is not tagged or released yet. APIs may change. Review every
> transformation diff and run your full test suite before materializing changes.

## Why BatchWeaver?

Batching can reduce repeated backend round trips, but a rewrite can also change
evaluation order, errors, cancellation, transactions, authorization, and result
mapping. BatchWeaver requires proof obligations for supported strategies,
rejects unknowns conservatively, leaves source unchanged by default, previews a
deterministic diff, and runs transformed tests through a Go overlay.

After the public tag exists, the supported install path will be:

```bash
go install github.com/Voskan/BatchWeaver/cmd/batchweaver@v0.1.0-beta.1
batchweaver version
batchweaver doctor
batchweaver scan ./...
```

See the [documentation site source](site/index.html),
[five-minute release quickstart](docs/release/release-notes-0.1.0-beta.1.md),
[compatibility matrix](docs/release/compatibility.md), and
[known issues](KNOWN-ISSUES.md).

## Development status

BatchWeaver is under active development. The repository now contains the
foundational semantic layer:

- the operation domain model (IDs, symbols, semantics, and contracts);
- typed scalar and batch function contracts and typed declarations;
- a strict, versioned (schema 1) YAML/JSON configuration system with includes,
  deterministic merge, and a semantic digest;
- a deterministic, machine-readable diagnostics system;
- configuration and operation CLI commands;
- an explicit, typed request-coalescing **runtime** (`runtime`, imported as
  `batchruntime`) that safely coalesces compatible scalar requests into bounded
  batch provider calls while preserving scope isolation, partition boundaries,
  independent cancellation, caller deadlines, per-item outcomes, and
  deterministic lifecycle behavior;
- a **static analysis** engine (`batchweaver scan`) that loads real Go modules,
  resolves operation declarations to canonical symbols, builds SSA and a
  conservative call graph, summarizes observable effects, discovers scalar
  operation call sites and their structural context (loops, goroutine fan-out),
  and exports a deterministic candidate inventory — without modifying source;
- a **semantic proof engine** (`batchweaver prove`) that turns each discovered
  candidate into a strategy-specific eligibility decision — proven eligible,
  proven ineligible, requires assumption, unknown, or deferred — derived from a
  closed registry of Go-semantic and operation-contract proof obligations, with
  deterministic, evidence-backed proof certificates and no source changes;
- a **transformation engine** (`batchweaver transform`, `build`, `test`, `run`)
  that consumes those certificates and performs static slice/array loop prefetch
  for a certified read-only operation — producing a deterministic plan, unified
  diff, and source map, building and testing through a Go overlay without editing
  source, and optionally materializing and reverting the edits safely;
- **runtime call lowering** that rewrites certified scalar calls into a typed,
  reflection-free runtime bridge (`bridge.Operation.Call`), coalescing compatible
  standalone, sibling, and existing goroutine/errgroup fan-out calls through the
  runtime while preserving context, cancellation, deadlines, partitions, and
  error semantics — with an overlay-first `-toolexec` integration, explicit
  batching barriers, and a guaranteed direct scalar fallback;
- a **backend adapter SDK** (`batchweaver adapter`) with a versioned manifest and
  capability model, a narrow real SQL parser that safely synthesizes exact-key
  PostgreSQL read batches (parameterized `unnest(...) WITH ORDINALITY` joins) or
  rejects unsupported queries with exact diagnostics, a typed reflection-free
  `database/sql` batch provider, Redis cluster hash-slot grouping and MGET/HMGET/
  pipeline mapping, and scalar/batch contract verification;
- **network protocol adapters** (`batchweaver graphql|grpc|http|openapi`): a fully
  implemented HTTP explicit-batch adapter over `net/http` with typed keyed/
  positional JSON envelopes and OpenAPI 3.1+ `x-batchweaver` binding; GraphQL
  resolver-wave analysis (a real, non-regex query parser, one scope per operation,
  selection/authorization partitioning, error/nullability preservation); an
  explicit gRPC batch-binding and metadata-partition policy layer; and protocol
  contract verification;
- an **adaptive scheduling and production tuning** layer (`batchweaver profile`,
  `tune`, `fairness`, `overload`, `wave`, `recursive`): privacy-safe, versioned
  workload profiles that store only bounded histograms and anonymized counts
  (never raw keys, tenants, or payloads); a versioned, explicitly weighted cost
  model; a bounded, explainable controller that recommends — and, only when
  explicitly enabled and within authoritative hard bounds, applies — `max_wait`,
  `max_batch_size`, concurrency, chunk size, and execution mode, with shadow and
  active modes, SLO guardrails, and automatic rollback; multi-operation wave
  planning; recursive breadth-first batching for proven traversals; fairness,
  quotas, and reserved capacity; overload detection, admission control, and
  non-silent load shedding; and deterministic offline replay, simulation, and
  tuning reports. Adaptive tuning can never bypass a semantic proof, exceed a
  configured bound, or guarantee a universal performance improvement;
- an **editor and developer-experience layer**: a standalone `batchweaver lsp`
  language server (LSP 3.17, no gopls internal imports) with an optional
  `--proxy-gopls` mode that launches and composes the user's gopls, a local
  workspace daemon (`batchweaver daemon`), and `batchweaver editor doctor`.
  BatchWeaver analyzes unsaved editor buffers through overlays and publishes live
  batching diagnostics, hover, code lenses, and preview code actions, with a VS
  Code extension (source) and standard-LSP setup for Neovim, Emacs/Eglot, Helix,
  and Zed. It never writes source implicitly, never collects telemetry, and is
  not a gopls plugin.
- a **non-publishing release assurance layer** (`batchweaver release`,
  `compatibility report`, `verify differential`, and `security audit`) that
  creates deterministic cross-platform snapshot archives, checksums, SPDX and
  CycloneDX SBOMs, unsigned local provenance, and a strict release manifest;
  verifies archive contents and digests offline; and rebuilds declared artifacts
  for byte comparison. Snapshot commands do not contain publication behavior.

**Arbitrary SQL transformation, automatic write synthesis, GraphQL/gRPC fusion,
and universal performance improvements are not implemented.** Only a narrow,
documented exact-key PostgreSQL read shape is synthesized; everything else is
rejected with an exact diagnostic. Every lowered operation still requires an
explicitly declared, compatible batch provider, and the concrete pgx, go-redis,
gqlgen, and grpc-go client bindings are contract-defined but deferred in this
build. Universal GraphQL optimization, remote batch-method generation, and
arbitrary HTTP request fusion are not implemented. See
[docs/guides/plan-a-transformation.md](docs/guides/plan-a-transformation.md),
[docs/guides/enable-runtime-lowering.md](docs/guides/enable-runtime-lowering.md),
[docs/guides/configure-database-sql.md](docs/guides/configure-database-sql.md),
[docs/reference/sql-support-matrix.md](docs/reference/sql-support-matrix.md),
[docs/limitations/backend-adapters.md](docs/limitations/backend-adapters.md),
[docs/limitations/network-adapters.md](docs/limitations/network-adapters.md),
[docs/limitations/adaptive.md](docs/limitations/adaptive.md), and
[docs/limitations/editor.md](docs/limitations/editor.md).

## The idea

Scalar code that issues one call per item is easy to write but often
inefficient, because each call pays a full round trip. BatchWeaver's goal is to
let you keep writing scalar code while it arranges for the underlying work to be
executed in batches.

The following is a simplified illustration. Current implementations are limited
to documented proven shapes and require an explicit compatible batch provider:

```go
// You write ordinary scalar code:
for _, id := range ids {
    user := LoadUser(ctx, id) // one logical call per id
    use(user)
}

// BatchWeaver's goal is to execute this as an equivalent batched operation:
users := LoadUsersBatch(ctx, ids) // one batched call for all ids
```

The transformation must be semantically transparent: observable behavior stays
the same, while redundant per-item work is coalesced.

## Planned architecture (high level)

BatchWeaver is planned as a pipeline that flows in one direction:

```text
CLI → project discovery → configuration → package loading → static analysis
→ intermediate representation → optimization planning → transformation
→ generated typed bindings → runtime scheduler → adapters → verification
→ observability
```

See [docs/architecture/overview.md](docs/architecture/overview.md) for details
and [ROADMAP.md](ROADMAP.md) for the phased plan.

## Requirements

- Go 1.26.5. Other toolchain versions are not supported by the current release
  policy until they are explicitly tested. With the default `GOTOOLCHAIN=auto`,
  the pinned toolchain can be fetched automatically.

## Build and test

```bash
# Build the CLI to bin/batchweaver
make build

# Run the current command
./bin/batchweaver version

# Or run without building:
go run ./cmd/batchweaver version

# Validate and inspect a configuration
go run ./cmd/batchweaver config validate --file examples/configuration/batchweaver.yaml
go run ./cmd/batchweaver operation list --file examples/configuration/batchweaver.yaml

# Run the full local quality gate
make check

# Build and independently verify an unpublished local snapshot
./bin/batchweaver release build --snapshot --output dist
./bin/batchweaver release verify dist/release-manifest.json
```

The selected beta is `v0.1.0-beta.1`; it has not been tagged or published because
mandatory authenticated publication gates remain blocked. See
[the launch decision](docs/release/0.1.0-beta.1/launch-decision.md),
[the compatibility report](docs/release/compatibility.md),
[known issues](KNOWN-ISSUES.md), and [release policy](docs/release/release-policy.md).

Example `config validate` output:

```text
Configuration is valid.
Schema: 1
Files: 1
Operations: 3
Digest: sha256:...
```

Example `version` output:

```text
BatchWeaver dev
Go: go1.26.5
Platform: darwin/arm64
Commit: unknown
Build date: unknown
```

## Repository structure

The layout and package boundaries are documented in
[docs/architecture/package-boundaries.md](docs/architecture/package-boundaries.md).

## Contributing

Contributions are welcome. Please read [CONTRIBUTING.md](CONTRIBUTING.md) before
opening a pull request.

## Security

Please report vulnerabilities privately as described in
[SECURITY.md](SECURITY.md). Do not open public issues for security reports.

## License

BatchWeaver is licensed under the Apache License, Version 2.0. See
[LICENSE](LICENSE) and [NOTICE](NOTICE).
