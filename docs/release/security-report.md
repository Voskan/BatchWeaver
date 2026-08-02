# Security Assurance Report

Scope: compiler inputs, proof trust, generated source, filesystem mutation,
daemon/LSP boundaries, SQL and OpenAPI parsing, protocol metadata, workload
profiles, workflows, and release artifact construction.

Repository-local audit checks tracked files for high-confidence private-key and
token patterns without printing matched values, verifies immutable action pins
and explicit workflow permissions, rejects unsafe archive paths and symlinks,
uses bounded manifest decoding, disables remote OpenAPI references, and provides
no publishing implementation. `make check` runs vet, golangci-lint, and pinned
govulncheck; CodeQL and dependency review run in GitHub workflows.

No private signing key is stored. Snapshot provenance is an unsigned local
in-toto/SLSA-compatible statement and is explicitly not a hosted-builder
attestation or a SLSA level claim. Keyless CI signing is a future authorized
publication concern. Sensitive exploit detail, if found, belongs in a private
GitHub Security Advisory.

Residual risks are recorded in `KNOWN-ISSUES.md`. No runtime telemetry or remote
collection is added by the release layer.
