# Roadmap

BatchWeaver plans work from verified defects, compatibility evidence, and user
needs. The roadmap is directional, has no promised dates, and does not convert
experimental features into support commitments.

## Current: observe and harden the first beta

- keep every published prerelease immutable and preserve its verified public
  assets; `v0.1.0-beta.2` supersedes beta.1 for new installations;
- monitor the Go module proxy, pkg.go.dev, release assets, documentation, and
  security-reporting paths;
- collect reproducible correctness, compatibility, installation, editor,
  adapter, and performance feedback without treating silence as success.

## Next: stabilize the supported surface

- reproduce and fix every verified P0/P1 beta defect with regression coverage;
- review the root, `operation`, `runtime`, `bridge`, `config`, and `diagnostics`
  APIs and approve a compatibility baseline;
- implement and test prerelease-to-v1 config, cache, generated-code, bridge,
  profile, daemon, and editor migration paths;
- extend the compatibility matrix with repeated public installs and supported Go
  patch releases;
- complete long-running race, fuzz, leak, soak, fault, and reproducibility
  campaigns on the release commit;
- add concrete integrations only when their client-version matrix can be tested
  and maintained.

## Stable v1 decision

Stable `v1.0.0` requires every mandatory exit criterion in
[`docs/release/v1.0.0/stable-release-decision.md`](docs/release/v1.0.0/stable-release-decision.md).
No date or popularity threshold substitutes for correctness, migration,
security, compatibility, installation, and governance evidence.

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
