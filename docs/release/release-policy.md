# Release Policy

BatchWeaver follows Semantic Versioning. Before 1.0, minor releases may contain
reviewed breaking changes; patch releases preserve supported public APIs and
versioned artifact decoders. Deprecations should remain for at least one minor
release unless retaining them would create a security or semantic-safety defect.

The first recommended candidate is `0.1.0-rc.1`. No tag has been selected or
published. The recommendation reflects the absence of prior tags and the broad
but pre-stable feature surface. Stable 1.0 requires a proven compatibility
window, supported concrete adapter bindings, repeated clean release runs, and
maintainer approval.

Go 1.26.5 is the sole supported toolchain for this candidate. Security patch
updates within Go 1.26 require a reviewed dependency/toolchain change. Supported
platforms and integrations are exactly those in `release/compatibility.json`.
An untested combination is never promoted to supported-and-tested.

Configuration, runtime ABI, proof, transform, daemon, LSP extension, and release
manifest versions are compatibility surfaces. Decoders reject unknown versions
unless a deterministic migration is explicitly documented and tested.

Release authority belongs to the maintainer identified by CODEOWNERS. Snapshot
commands cannot publish. Tags, GitHub Releases, Marketplace packages, and package
manager changes require explicit approval in the release execution context.
Security releases use GitHub Security Advisories as documented in `SECURITY.md`.
Artifacts are retained according to the hosting channel's policy; superseded or
bad artifacts follow the rollback plan.
