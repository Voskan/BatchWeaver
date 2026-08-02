# Release Policy

BatchWeaver follows Semantic Versioning and never moves, rewrites, or reuses a
published tag. Versions below v1 are prerelease and may make reviewed breaking
changes with migration notes. A stable v1 release begins the compatibility
policy described in the API-freeze decision.

## Current version

The current stable release is `v1.0.0`. It is distributed by an immutable Git
tag and a GitHub Release, including platform archives, checksums, SBOMs, local
provenance statements, and a VSIX. The Go module uses the same tag. The
`v0.1.0-beta.1`, `v0.1.0-beta.2`, and `v0.1.0-beta.3` prereleases remain
published and immutable as historical evidence.

Publication requires green mandatory workflows, verified repository identity and
permissions, checksums, SBOMs, documented provenance limitations, installation
tests, security reporting, rollback, and a factual release decision. Snapshot
commands cannot publish; the separately confirmed publish helper uploads only
manifest-declared assets from the exact immutable tag.

`v1.0.0` artifacts are unsigned and carry no hosted attestation; that gap is
recorded as an accepted risk with a remediation plan in the stable-release
decision and in `KNOWN-ISSUES.md`.

## Stable series

`v1.0.0` freezes the Tier 1 public Go API under Semantic Versioning; `bridge`
and the `adapters/*` packages are experimental and excluded from that promise.
The tiers, deprecation policy, and supported Go window are in the
[API freeze](v1.0.0/api-freeze.md). Every criterion, including the four accepted
risks, is recorded in the
[stable-release decision](v1.0.0/stable-release-decision.md) and the
machine-readable gate report, whose decision must match its gate states.

Within `v1`, a patch release fixes defects without API change, and a minor
release may add Tier 1 API or change Tier 2 packages. Removing a Tier 1
identifier requires a new major version.

## Compatibility

Go 1.26.x is the tested toolchain window for the current candidate: 1.26.0 is
the minimum and 1.26.5 is the current pin. Supported platforms and integrations
are exactly those in `release/compatibility.json`.
Untested combinations remain untested.

Configuration, diagnostic JSON, bridge ABI, proof, transform, daemon, profile,
adapter, editor, and release-manifest versions are independent compatibility
surfaces. Unknown versions fail closed unless a tested migration explicitly
supports them.

## Authority and channels

Release authority belongs to the maintainer identified by CODEOWNERS and
GOVERNANCE.md. Tags, GitHub Releases, module-proxy publication, Marketplace
packages, Pages deployment, and package-manager metadata require explicit
release approval and verified credentials. Security releases use the private
advisory process in SECURITY.md.

Post-v1 patch, minor, security, dependency, backport, and retention procedures
are defined in [maintainer release procedures](../maintainers/release-procedures.md).
