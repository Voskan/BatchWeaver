# Roadmap

BatchWeaver plans work from verified defects, compatibility evidence, and user
needs. The roadmap is directional, has no promised dates, and does not convert
experimental features into support commitments.

## Current: close the v1.0.0 accepted risks

`v1.0.0` is published with four documented accepted risks. Closing them is the
immediate priority:

- publish **signed artifacts with hosted build attestation** in the next patch
  release, replacing the current checksum-and-local-provenance-only integrity;
- attach **hosted compatibility and production-campaign artifacts** produced at
  an exact released commit;
- add **live PostgreSQL and Redis Cluster acceptance** alongside the existing
  hermetic client coverage;
- keep every published version immutable, monitor the module proxy, pkg.go.dev,
  release assets, documentation, and security-reporting paths, and triage
  reproducible reports without treating silence as success.

## Next: broaden the supported surface

- reproduce and fix every verified P0/P1 report with regression coverage;
- extend the compatibility matrix with repeated public installs and supported Go
  patch releases;
- promote artifact schemas from `v1alpha1` once their formats have stabilized;
- decide whether `bridge` and the `adapters/*` packages graduate from the
  experimental tier, based on their client-version matrices;
- add concrete integrations only when their client-version matrix can be tested
  and maintained.

## Compatibility commitments

The Tier 1 Go API is frozen for the `v1` series under Semantic Versioning; the
exact tiers, deprecation policy, and supported Go window are in
[`docs/release/v1.0.0/api-freeze.md`](docs/release/v1.0.0/api-freeze.md). The
accepted risks and their remediation plans are in
[`docs/release/v1.0.0/stable-release-decision.md`](docs/release/v1.0.0/stable-release-decision.md).

## Potential post-v1 themes

These are research and ecosystem directions, not commitments:

- additional database, cache, GraphQL, gRPC, and HTTP adapters;
- wider proof-supported transformation shapes;
- deeper gopls/editor workflows with versioned edit application;
- profile exporters and bounded-label observability integrations;
- compiler and runtime performance work backed by reproducible benchmarks;
- distributed batching research that preserves explicit isolation contracts.

New major features require public design review and an ADR. Patch, minor,
security, deprecation, and backport policy is documented in
[`docs/maintainers/release-procedures.md`](docs/maintainers/release-procedures.md).
