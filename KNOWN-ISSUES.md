# Known Issues

These are verified limitations and release blockers for the selected,
unpublished `v0.1.0-beta.1` candidate. No stable release is implied.

## BW-KI-001 — Concrete framework client bindings are absent

- Severity: P2
- Affected: pgx, go-redis, gqlgen, and grpc-go integrations
- Impact: framework-neutral contracts and verification logic exist, but users
  must provide an explicit binding.
- Workaround: use the Adapter SDK, `database/sql`, or `net/http` providers.
- Stable blocker: yes until the supported v1 surface deliberately includes or
  excludes each integration.

## BW-KI-002 — VS Code host end-to-end evidence is absent

- Severity: P1 for Marketplace publication
- Impact: lint, typecheck, unit tests, package construction, and VSIX contents
  pass locally, but installation and activation in a real Extension Host have
  not been verified.
- Workaround: use the standalone LSP and test the VSIX in a disposable Extension
  Development Host.
- Stable blocker: yes for a supported VS Code release.

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

## BW-KI-005 — Hosted governance and security settings are unverified

- Severity: P1 for publication
- Impact: the public API confirms the repository identity but unauthenticated
  access cannot verify branch protection, required checks, environments, private
  vulnerability reporting, release authority, or Pages configuration.
- Workaround: an authorized repository owner must verify the settings before
  publication.
- Stable blocker: yes.

## BW-KI-006 — Dependency Graph is disabled

- Severity: P1 for release assurance
- Evidence: PR #7 Dependency Review run `30739695826` failed with
  “Dependency review is not supported on this repository.”
- Impact: the mandatory dependency-change policy cannot execute.
- Workaround: enable Dependency Graph in repository Security settings and rerun
  the failed workflow. Do not mark the check optional.
- Stable blocker: yes.

## BW-KI-007 — Windows line-ending correction needs hosted verification

- Severity: P1 for Windows support
- Evidence: PR #7 CI run `30739695835` failed because `.txt` golden baselines
  were checked out as CRLF while generated output used LF.
- Remediation: `.gitattributes` now enforces LF for `.txt` files.
- Stable blocker: yes until a hosted Windows run passes at the release commit.

## BW-KI-008 — No public module or release artifacts exist

- Severity: P1 for installation
- Impact: `go get`/`go install` at `v0.1.0-beta.1`, pkg.go.dev documentation,
  public checksums, public SBOMs, and archive installation cannot be verified.
- Workaround: repository evaluation from a reviewed commit only.
- Stable blocker: yes.

## BW-KI-009 — Stable evidence period has not begun

- Severity: P1 for v1 governance
- Impact: there are no published prereleases, external compatibility reports,
  verified downstream integrations, or public installation results. Absence of
  reports is not stability evidence.
- Workaround: publish a gated beta, collect and reproduce evidence, then repeat
  the stable-release audit.
- Stable blocker: yes.
