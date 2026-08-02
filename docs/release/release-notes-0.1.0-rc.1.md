# BatchWeaver 0.1.0-rc.1 Draft Release Notes

Status: unpublished release-candidate recommendation.

This candidate combines the typed batching runtime, conservative analysis and
proof engine, deterministic transformations, adapters, adaptive scheduling,
daemon/LSP tooling, and a non-publishing release assurance pipeline. Supported
environments are limited to the explicit compatibility report. Concrete pgx,
go-redis, gqlgen, and grpc-go bindings are not included.

Verify a downloaded snapshot offline with `batchweaver release verify
release-manifest.json`. Checksums, SPDX and CycloneDX SBOMs, and unsigned local
provenance are generated. Signatures are disabled for snapshots. Read
`KNOWN-ISSUES.md` before evaluation and report security issues privately through
GitHub Security Advisories.
