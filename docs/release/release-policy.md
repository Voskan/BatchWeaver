# Release Policy

BatchWeaver follows Semantic Versioning and never moves, rewrites, or reuses a
published tag. Versions below v1 are prerelease and may make reviewed breaking
changes with migration notes. A stable v1 release begins the compatibility
policy described in the API-freeze decision.

## Current version

The current public beta is `v0.1.0-beta.1`. It is distributed by an immutable
Git tag and prerelease GitHub Release, including platform archives, checksums,
SBOMs, local provenance statements, and a VSIX. The Go module uses the same tag.

Beta publication requires green mandatory workflows, verified repository
identity and permissions, checksums, SBOMs, provenance limitations, installation
tests, security reporting, rollback, and a factual release decision. Snapshot
commands cannot publish; the separately confirmed publish helper uploads only
manifest-declared assets from the exact immutable tag.

## Stable release

Stable `v1.0.0` requires all mandatory criteria in the
[stable-release decision](v1.0.0/stable-release-decision.md). In particular, a
published prerelease evidence period, supported-platform installation, migration
tests, API-freeze approval, security reporting, and governance approval cannot
be replaced by local test success.

## Compatibility

Go 1.26.5 is the sole tested toolchain for the current candidate. Supported
platforms and integrations are exactly those in `release/compatibility.json`.
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
