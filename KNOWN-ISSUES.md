# Known Issues

This file lists verified limitations for the proposed `0.1.0-rc.1` snapshot. It
does not imply that the candidate has been published.

## BW-KI-001 — Concrete framework client bindings are absent

- Severity: major
- Affected: pgx, go-redis, gqlgen, and grpc-go integrations
- Impact: the framework-neutral contracts and verification logic exist, but
  users must write an explicit binding; documentation must not claim turnkey
  client integration.
- Workaround: use the Adapter SDK, `database/sql`, or `net/http` providers.
- Tracking: roadmap, post-RC integration milestone
- Release blocker: no, when the release notes and support matrix remain explicit
- Disclosure: public

## BW-KI-002 — VS Code host end-to-end test is not automated locally

- Severity: major
- Affected: VS Code extension installation and activation
- Impact: lint, typecheck, manifest consistency, and VSIX contents are tested,
  but a real headless Extension Host session has not been demonstrated.
- Workaround: use the standalone LSP or install the local VSIX in a disposable
  Extension Development Host before relying on it.
- Tracking: Prompt 13 launch prerequisites
- Release blocker: yes for Marketplace publication; no publication is authorized
- Disclosure: public

## BW-KI-003 — Shared daemon cache is not used by CLI/LSP analysis

- Severity: minor
- Affected: repeated editor and CLI analysis
- Impact: correct results, but avoidable repeated analysis and memory use.
- Workaround: run commands normally; no semantic behavior is affected.
- Tracking: roadmap
- Release blocker: no
- Disclosure: public

## BW-KI-004 — LSP transformation application remains preview-first

- Severity: minor
- Affected: editor transformation workflow
- Impact: the editor previews a transformation and directs users to the CLI;
  it does not apply a full version-preconditioned WorkspaceEdit.
- Workaround: use `batchweaver transform diff` and `materialize`.
- Tracking: roadmap
- Release blocker: no
- Disclosure: public

## BW-KI-005 — Hosted branch-protection state was not authenticated

- Severity: major
- Affected: GitHub release authorization and required-check enforcement
- Impact: CODEOWNERS, least-privilege workflows, and repository policies are
  reviewable in Git, but the GitHub branch-protection API returned 401 without an
  authenticated maintainer token, so hosted enforcement was not verified.
- Workaround: an authorized maintainer must inspect main-branch rules and required
  checks before approving publication.
- Tracking: Prompt 13 launch checklist
- Release blocker: yes for public publication; snapshot construction remains local
- Disclosure: public

## BW-KI-006 — Superseded dependency update pull requests remain open

- Severity: informational
- Affected: GitHub Actions dependencies
- Impact: six Dependabot pull requests were open during the audit. Prompt 12
  independently verified and pinned current action releases, so those branches
  may now be stale or partially superseded.
- Workaround: review CI results and close or rebase pull requests 1–6; do not
  merge them mechanically over the reviewed pins.
- Tracking: GitHub pull requests 1–6
- Release blocker: no while current pinned checks pass without a known finding
- Disclosure: public
