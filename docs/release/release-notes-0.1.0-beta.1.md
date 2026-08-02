# BatchWeaver 0.1.0-beta.1 Release Notes

Status: selected public-beta version; publication is blocked until authenticated
GitHub ownership, protection, hosted checks, and release permissions are verified.

## Summary

BatchWeaver is a proof-gated Go compiler/runtime toolkit that turns supported
scalar access patterns into reviewed batch execution. This beta packages the
compiler, proof engine, overlay-first transformer, typed runtime, adapter
contracts, daemon/LSP, VS Code extension, and release assurance tooling.

## Why BatchWeaver

Manual batching can improve throughput but can also change ordering, errors,
cancellation, authorization boundaries, and transaction identity. BatchWeaver
requires a semantic proof for each supported transformation and rejects cases it
cannot justify conservatively.

## Included and supported

- static loop prefetch and certified runtime-call lowering for documented shapes;
- explicit typed batch providers and `database/sql`/`net/http` integrations;
- deterministic plans, diffs, overlays, transformed tests, materialize/revert;
- macOS arm64 native validation and five cross-platform archives;
- standalone LSP, optional gopls proxy, and a locally packaged VSIX.

## Installation

After publication:

```bash
go install github.com/Voskan/BatchWeaver/cmd/batchweaver@v0.1.0-beta.1
batchweaver version
batchweaver doctor
```

Archives must be checked against `SHA256SUMS` before execution. The VSIX can be
installed locally with
`code --install-extension batchweaver-vscode-0.1.0-beta.1.vsix`.

## Five-minute quickstart

```bash
git clone https://github.com/Voskan/BatchWeaver.git
cd BatchWeaver
git checkout v0.1.0-beta.1
go run ./cmd/batchweaver scan ./examples/static-prefetch/...
go run ./cmd/batchweaver prove ./examples/static-prefetch/...
go test -race ./examples/static-prefetch/...
```

Preview and test before any explicit materialization. Source remains unchanged
by default.

## Compatibility and known limitations

Go 1.26.5 is the supported toolchain. Native evidence is strongest on
macOS/arm64; other artifact targets have limited cross-build evidence as recorded
in `release/compatibility.json`. Concrete pgx, go-redis, gqlgen, and grpc-go
bindings are not included. The VSIX has not completed a real Extension Host E2E.
See `KNOWN-ISSUES.md`.

## Breaking-change policy

This is a beta. Public Go APIs, schemas, runtime ABI, and generated artifacts may
change between prereleases with changelog and migration guidance. Tags and
released assets are immutable.

## Upgrade, rollback, and verification

Keep tool/runtime/extension versions aligned, regenerate plans after upgrades,
and invalidate versioned caches. Roll back by disabling transformations, using
the scalar path, reverting materialized changes, clearing versioned caches, and
installing the prior version. Verify artifacts with `sha256sum -c SHA256SUMS` or
`shasum -a 256 -c SHA256SUMS`, then run `batchweaver release verify` when the
manifest and complete asset set are together.

## Security and feedback

Do not report vulnerabilities or transformation isolation failures in a public
issue; follow `SECURITY.md`. Reproducible bugs and beta feedback use the GitHub
issue forms. BatchWeaver uploads no source or runtime data and adds no telemetry.

## Contributors

This prerelease preparation was produced from the repository history. No external
adoption, contributor, or production-validation claim is made.
