# Known Issues

These are verified limitations for the public `v0.1.0-beta.3` beta. No stable
release is implied.

## BW-KI-011 — beta.2 downloaded checksum set uses unavailable report paths

- Severity: P1 for `v0.1.0-beta.2`; fixed in `v0.1.0-beta.3`
- Affected: complete asset sets downloaded from the beta.2 GitHub Release
- Impact: GitHub flattened five `reports/...` assets to their base names while
  the published `SHA256SUMS` retained the directory prefixes, so whole-set
  checksum verification reports those five paths missing.
- Workaround: install `v0.1.0-beta.3`. Individual beta.2 file digests remain
  available from GitHub, but beta.2 is not recommended for new installations.
- Stable blocker: resolved by a flat, unique asset layout enforced by tests.

## BW-KI-010 — beta.1 `go install` reports a development version

- Severity: P1 for `v0.1.0-beta.1`; fixed in `v0.1.0-beta.2`
- Affected: binaries built with `go install ...@v0.1.0-beta.1`
- Impact: the module installs correctly, but `batchweaver version` reports
  `dev` because beta.1 did not derive version metadata from Go build info.
- Workaround: install `v0.1.0-beta.3` or use a beta.1 release archive, whose
  release builder injected the correct metadata.
- Stable blocker: resolved; regression coverage is required.

## BW-KI-001 — Concrete framework client bindings are absent

- Severity: P2 for `v0.1.0-beta.3`; resolved on the default branch
- Affected: pgx, go-redis, gqlgen, and grpc-go integrations in published beta.3
- Resolution: public `adapters/pgxv5`, `adapters/redisv9`, `adapters/gqlgen`,
  and `adapters/grpcgo` packages add typed client integrations and hermetic
  integration coverage.
- Workaround for beta.3: use the Adapter SDK, `database/sql`, or `net/http`
  providers, or evaluate the reviewed source commit containing the bindings.
- Stable blocker: focused implementation is resolved; broader client/platform
  compatibility evidence remains required.

## BW-KI-002 — VS Code distribution is VSIX-only

- Severity: P2
- Impact: the VSIX installs and activates in a real VS Code Extension Host, but
  it is not listed in the Visual Studio Marketplace.
- Workaround: install the verified VSIX from the GitHub Release assets.
- Stable blocker: marketplace distribution is not required; broader editor
  compatibility evidence remains required before v1.

## BW-KI-003 — Shared daemon cache is not used by CLI/LSP analysis

- Severity: P3
- Impact: results remain correct, but repeated analysis can consume avoidable
  time and memory.
- Workaround: run commands normally; no semantic behavior is affected.
- Stable blocker: no when documented.

## BW-KI-004 — Editor transformation application is preview-first

- Severity: P3
- Impact: the editor previews transformations and directs users to the CLI; it
  does not apply a full version-preconditioned `WorkspaceEdit`.
- Workaround: use `batchweaver transform diff` and explicit materialization.
- Stable blocker: no when documented.

## BW-KI-005 — Hosted provenance is unavailable

- Severity: P2 for beta, P1 for a stronger stable supply-chain claim
- Impact: release manifests contain local unsigned provenance statements, not a
  hosted SLSA attestation.
- Workaround: verify the immutable tag, `SHA256SUMS`, SBOMs, and reproducible
  artifacts.
- Stable blocker: policy decision required before v1.

## BW-KI-006 — Beta artifacts and tag are unsigned

- Severity: P2 for beta
- Impact: integrity is protected by immutable GitHub assets and SHA-256
  checksums, but no configured signing identity exists.
- Workaround: compare downloads with `SHA256SUMS` and verify the tag commit.
- Stable blocker: signing policy must be explicitly resolved before v1.

## BW-KI-008 — Go/toolchain compatibility is narrow

- Severity: P2
- Impact: Go 1.26.5 is the only supported toolchain for this beta; additional
  Go and integration client versions remain untested.
- Workaround: use the declared toolchain and compatibility matrix.
- Stable blocker: broader approved compatibility evidence is required.

## BW-KI-009 — Stable evidence period has not begun

- Severity: P1 for v1 governance
- Impact: the first published prerelease has no completed evidence period,
  verified downstream integrations, or external compatibility reports. Absence
  of reports is not stability evidence.
- Workaround: collect and reproduce beta evidence, then repeat the stable-release
  audit.
- Stable blocker: yes.
