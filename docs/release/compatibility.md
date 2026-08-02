# Compatibility Report

The machine-readable policy is [`release/compatibility.json`](../../release/compatibility.json).
The blocking hosted implementation is
[`Compatibility Matrix`](../../.github/workflows/compatibility.yml). A support
claim is valid only when the `Compatibility policy` check succeeds for the exact
commit and its `compatibility-run-<commit>` evidence artifact remains available.

## Go support window

The minimum supported toolchain is `go1.26.0`; the current release toolchain is
`go1.26.5`. Both run the full suite with `GOTOOLCHAIN=local`, so the minimum job
cannot silently download and substitute the newer toolchain. The supported
window is `go1.26.x`. Older, development, and future minor toolchains are not
claimed until a reviewed matrix row exists.

The current pin follows the [official Go release history](https://go.dev/doc/devel/release).
Pins change only through a reviewed pull request. A newer patch does not become
supported merely because it exists.

## Hosted dimensions

Every pull request, `main` push, weekly schedule, and manual dispatch runs these
independent jobs:

- full tests and CLI builds on minimum and current Go;
- native integration subsets on hosted Linux, macOS, and Windows runners, with
  the actual runner OS/architecture recorded;
- all-package, CGO-disabled cross-builds for every published target:
  linux/amd64, linux/arm64, darwin/amd64, darwin/arm64, and windows/amd64;
- full readonly-module, generated-vendor, and native-CGO suites, plus a generated
  go.work workspace-wide list/build and core runtime/operation/adapter suite;
- public-API fixtures for pgx v5.10.0, go-redis v9.21.0, gqlgen v0.17.94, and
  grpc-go v1.83.0;
- a real gopls v0.21.1 child process through proxy initialize, capability merge,
  shutdown, and exit; and
- real VS Code 1.85.2 (minimum) and 1.131.0 (current) Extension Hosts, checking
  extension activation and command registration.

Hermetic client tests do not imply a live PostgreSQL or multi-node Redis Cluster
deployment. Community editor configuration does not imply a maintained UI/E2E
matrix. Those boundaries remain explicit in the machine-readable rows.

## Evidence contract

Each matrix job uploads a
`batchweaver.compatibility-evidence/v1alpha1` JSON record containing the commit,
hosted run URL, workflow ref, actual runner OS/architecture, Go version,
dimension, tested value, command description, status, and time. The aggregate
job rejects missing rows, wrong commits, non-success states, wrong repository run
URLs, and any count other than 18, then uploads one
`batchweaver.compatibility-run/v1alpha1` artifact retained for 90 days.

`scripts/verify-hosted-compatibility.sh` downloads and verifies that artifact for
the exact tag commit. Release publication invokes this check and refuses to
continue when hosted evidence is missing, expired, unsuccessful, or belongs to a
different commit.
