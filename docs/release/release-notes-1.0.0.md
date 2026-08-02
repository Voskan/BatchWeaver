# BatchWeaver v1.0.0

The first stable release. `v1.0.0` freezes the public Go API under Semantic
Versioning and ships a tested upgrade path from every published prerelease.

## What BatchWeaver is

A proof-gated batching compiler and typed request-coalescing runtime for Go. It
finds supported scalar access patterns — including common N+1 query shapes —
proves their safety conditions, previews deterministic changes, and executes
compatible calls in bounded batches without silently crossing request, tenant,
authorization, transaction, or session boundaries.

## Highlights since `v0.1.0-beta.3`

- **Concrete client bindings.** Public `adapters/pgxv5`, `adapters/redisv9`,
  `adapters/gqlgen`, and `adapters/grpcgo` packages with hermetic coverage
  (pgxmock, miniredis, the public gqlgen extension API, and grpc-go bufconn).
- **Composite and bounded-join SQL synthesis.** Composite-key PostgreSQL reads
  and one explicitly at-most-one INNER/LEFT join, with integrity validation.
- **Shared workspace analysis cache.** The daemon owns a bounded,
  content-addressed cache that CLI and editor requests reuse.
- **Hosted compatibility matrix enforcement** and a scheduled production-like
  campaign covering eleven fuzz categories, soak, leak budgets, and faults.
- **Executable prerelease migration suite** from beta.1, beta.2, and beta.3.

## Compatibility

- **Tier 1 (stable, SemVer):** module root, `config`, `diagnostics`,
  `operation`, `runtime`.
- **Tier 2 (experimental):** `bridge` and the four `adapters/*` packages. They
  ship in the `v1` module but may change incompatibly in a minor release.
- **Artifact schemas** remain `v1alpha1` and are regenerated, not migrated.
- **Go:** `go1.26.x`; minimum `go1.26.0`, current `go1.26.5`.

Full detail: [API freeze](v1.0.0/api-freeze.md).

## Install

```bash
go install github.com/Voskan/BatchWeaver/cmd/batchweaver@v1.0.0
go get github.com/Voskan/BatchWeaver@v1.0.0
```

## Upgrading from a beta

Configuration schema 1 is unchanged, so beta configuration loads as-is. Discard
analysis, proof, transform, profile, daemon, and editor caches and regenerate
bridges and transformation plans rather than reusing beta artifacts. The full
sequence is in [migration](v1.0.0/migration.md), which is executable and runs on
every test run.

## Accepted risks

This release ships with four documented gaps. They are recorded, not hidden.

1. **Artifacts are unsigned** and carry no hosted build attestation. Integrity
   is verifiable through SHA-256 checksums, SPDX/CycloneDX SBOMs, a local
   provenance statement, and reproducible builds.
2. **Hosted compatibility evidence was not observed at the tagged commit.** The
   matrix passes locally and the workflow exists.
3. **The public prerelease period was short**, so field evidence is limited to
   the project's automated suites. No long-term production-stability claim is
   made.
4. **No live-backend acceptance.** Client integrations use hermetic fakes.

Each has a remediation plan in the
[stable-release decision](v1.0.0/stable-release-decision.md) and the
machine-readable [gate report](../../release/gates-v1.0.0.json).

## Verification

```bash
sha256sum --check SHA256SUMS
batchweaver version
batchweaver doctor
```

## Known limitations

See [KNOWN-ISSUES.md](../../KNOWN-ISSUES.md) and
[docs/limitations](../limitations/).
