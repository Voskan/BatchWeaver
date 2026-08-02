# Public API Freeze — v1.0.0

## Status

**Approved for v1.0.0.** The exported Go surface below is frozen under Semantic
Versioning for the `v1` major series, with the stability tiers stated here. The
canonical element-level inventory is `internal/release/testdata/public-api.txt`,
verified by `TestPublicAPIBaseline`; the machine-readable summary is
`release/api-inventory-v1.json`.

Go modules version a module, not individual packages. This document therefore
states, per package, what compatibility the project actually promises.

## Tier 1 — stable

Breaking changes require a new major version.

| Package | Records | Contract |
| --- | ---: | --- |
| module root | 60 | batch request/response, outcomes, typed declarations |
| `config` | 50 | configuration schema 1, strict unknown-key policy |
| `diagnostics` | 78 | diagnostic model, codes, severities, formatters |
| `operation` | 761 | operation identity, semantics, and policy contracts |
| `runtime` | 166 | scopes, binding, providers, partitions, statistics |

The `operation` and `runtime` surfaces are large. They are frozen as reviewed:
identifiers, signatures, and documented behaviour do not change incompatibly in
`v1`. Additive evolution (new options, new fields on option structs, new
methods) remains permitted.

## Tier 2 — experimental

These packages ship inside the `v1` module but are explicitly **not** covered by
the `v1` compatibility promise. They may change incompatibly in a minor release.
Every one of them is documented as experimental in its package documentation.

| Package | Records | Reason |
| --- | ---: | --- |
| `bridge` | 6 | generated-code ABI, versioned `batchweaver.bridge/v1alpha1` |
| `adapters/pgxv5` | 6 | tracks pgx v5.10.0 |
| `adapters/redisv9` | 11 | tracks go-redis v9.21.0 |
| `adapters/gqlgen` | 9 | tracks gqlgen v0.17.94 |
| `adapters/grpcgo` | 11 | tracks grpc-go v1.83.0 |

The four client packages depend on third-party client APIs whose own evolution
BatchWeaver does not control. Applications that need the stable tier should
build providers against the Adapter SDK, `database/sql`, or `net/http` instead.

## Artifact schemas

Compiler and runtime artifacts are versioned independently of the Go API and
remain `v1alpha1`. They are not a Go compatibility surface: they are regenerated
rather than migrated, and an incompatible schema, ABI, contract, source anchor,
or toolchain invalidates the corresponding cache.

- analysis, proof, and transformation snapshots;
- adapter manifests and verification contracts;
- profile, controller, wave, and replay artifacts;
- daemon protocol `batchweaver.daemon/v1alpha1`;
- bridge ABI `batchweaver.bridge/v1alpha1`.

Promoting any of these to a stable `v1` schema is a future minor-release task
and does not affect the frozen Go API.

## Other compatibility surfaces

- CLI command hierarchy and flags: covered by `batchweaver help` and the CLI
  reference; commands and flags are additive within `v1`.
- Exit codes: `docs/reference/exit-codes.md`; existing codes keep their meaning.
- Diagnostic codes: `docs/reference/diagnostic-codes.md`; a published code keeps
  its meaning and range.
- Configuration: schema 1; a future schema 2 must load schema 1.

## Compatibility window

- **Supported Go:** `go1.26.x`; minimum `go1.26.0`, current `go1.26.5`. A future
  minor release may raise the minimum to the two most recent Go releases, which
  is not treated as a breaking change to the Go API.
- **Deprecation:** a Tier 1 identifier scheduled for removal is documented as
  deprecated in at least one minor release before a major release removes it.
- **Security:** a security fix may change behaviour within `v1` when no
  compatible fix exists; the change is documented in the release notes and
  `KNOWN-ISSUES.md`.

## Enforcement

`TestPublicAPIBaseline` fails when the exported surface drifts from the recorded
inventory, so an unintended addition or removal cannot reach a release
unnoticed. Updating the baseline is an explicit, reviewed change.

## Approval

Approved by the repository maintainer and owner for `v1.0.0`. The accepted risks
that accompany this release are recorded in
[stable-release-decision.md](stable-release-decision.md).
