# Stable v1.0.0 Release Decision

## Decision

**BLOCKED — do not tag or publish `v1.0.0`.**

The selected version remains the unpublished `v0.1.0-beta.1`. Local assurance
cannot replace a public prerelease, installation, migration, compatibility, and
governance evidence period.

## Mandatory exit criteria

| Criterion | Status | Evidence or blocker |
| --- | --- | --- |
| No unresolved P0/P1 | blocked | dependency-policy, installation, editor E2E, and hosted settings remain P1 publication blockers |
| Supported transformations differential-tested | pass locally | deterministic differential suite; must rerun on final commit |
| Mandatory mutations killed | pass locally | 12/12 modeled critical mutations; final rerun required |
| Compatibility matrix | partial | Linux, macOS, and Windows hosted jobs pass; only Go 1.26.5 and limited integrations are tested |
| Upgrade from supported prereleases | blocked | no prerelease has been published |
| Installation | blocked | no tag, proxy version, release asset, or public docs deployment |
| Race, fuzz, and security suites | partial | local evidence exists; dependency review cannot run |
| Artifacts verifiable | partial | local artifacts verify; no public assets or hosted attestations |
| Documentation complete | partial | source audit improved; public site and user feedback unavailable |
| Public API freeze approved | blocked | inventory exists; no compatibility window or approval |
| Security reporting works | blocked | private vulnerability reporting setting unverified |
| Rollback/hotfix procedure works | pass locally | documented and script-gated; public rehearsal pending |
| Known limitations published | pass | `KNOWN-ISSUES.md` |
| Release dry run | pass locally | clean snapshot/reproduction evidence; final rerun required |
| Public prerelease evidence sufficient | blocked | no prerelease, downloads, installs, or external reports |
| Governance approval | blocked | no explicit stable-release approval |

## Accepted risks

None are accepted for stable publication. Documented beta limitations remain
eligible only for an approved prerelease decision.

## Rollback

Default source mutation and active adaptive tuning remain off. Users can retain
scalar execution, use overlays, revert materialization through the backup
manifest, invalidate incompatible caches, and install a patched version. See
`docs/release/rollback.md`.

## Continuation

1. fix and observe green PR #7 checks;
2. enable and verify required GitHub security/governance settings;
3. publish and verify the beta only with release authority;
4. collect a real evidence period and resolve verified P0/P1 reports;
5. test migration from every published prerelease;
6. repeat all final gates and record maintainer approval.

There is no release commit, tag, GitHub stable release, stable documentation, or
stable package-manager publication associated with this decision.
