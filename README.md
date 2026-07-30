# BatchWeaver

BatchWeaver is an automatic semantic batching compiler and runtime for Go.

It aims to convert ordinary scalar Go calls into semantically equivalent,
optimized batch execution — without forcing you to restructure your code by
hand — through static analysis, compile-time transformation, generated typed
bindings, and a batching-aware runtime.

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
  that consumes those certificates and performs the first production
  transformation — static slice/array loop prefetch for a certified read-only
  operation — producing a deterministic plan, unified diff, and source map,
  building and testing through a Go overlay without editing source, and
  optionally materializing and reverting the edits safely.

**Broad Go coverage, SQL/backend batch-provider generation, write batching,
goroutine fan-out lowering, and universal performance improvements are not
implemented.** BatchWeaver transforms only a certified subset of read-only
slice/array loops; every other candidate class is discovered and proven but not
yet rewritten. See [docs/guides/runtime-api.md](docs/guides/runtime-api.md),
[docs/guides/scan.md](docs/guides/scan.md),
[docs/guides/proving-candidates.md](docs/guides/proving-candidates.md),
[docs/guides/plan-a-transformation.md](docs/guides/plan-a-transformation.md), and
[docs/limitations/prompt-06.md](docs/limitations/prompt-06.md).

## The idea

Scalar code that issues one call per item is easy to write but often
inefficient, because each call pays a full round trip. BatchWeaver's goal is to
let you keep writing scalar code while it arranges for the underlying work to be
executed in batches.

The following illustrates the **target** behavior. It is a conceptual example of
what BatchWeaver intends to enable — it is **not** implemented today:

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

- Go 1.26 or newer. The module pins `toolchain go1.26.5`; with the default
  `GOTOOLCHAIN=auto`, the correct toolchain is fetched automatically.

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
```

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
