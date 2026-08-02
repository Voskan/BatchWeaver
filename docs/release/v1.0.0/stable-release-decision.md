# Stable v1.0.0 Release Decision

## Decision

**BLOCKED — do not tag or publish `v1.0.0`.**

The selected version is the `v0.1.0-beta.3` public beta; beta.1 and beta.2
remain immutable historical evidence. The successful prerelease gates begin—rather than complete—the installation,
migration, compatibility, feedback, and governance evidence period required for
stable v1.

## Mandatory exit criteria

| Criterion | Status | Evidence or blocker |
| --- | --- | --- |
| No unresolved P0/P1 | pass for current verified evidence | Windows text normalization, dependency configuration, and VS Code command-registration defects were resolved; future reports must still be triaged |
| Supported transformations differential-tested | pass locally | deterministic differential suite; must rerun on final commit |
| Mandatory mutations killed | pass locally | 12/12 modeled critical mutations; final rerun required |
| Compatibility matrix | partial | Blocking minimum/current Go, OS/target, build-mode, exact-client, real-gopls, and real-VS-Code jobs exist; the combined hosted artifact must still pass for the exact final candidate commit |
| Upgrade from supported prereleases | blocked | migration from this beta to a future v1 candidate cannot yet be exercised |
| Installation | pass for current beta | beta.3 public Go proxy, version metadata, complete release checksum set, archives, and VSIX are post-publication-verified; v1 installation remains future work |
| Race, fuzz, and security suites | partial | race, bounded fuzz, CodeQL, vulnerability, secret, and Dependency Review evidence exists; final extended campaigns remain |
| Artifacts verifiable | partial | public beta assets, checksums, SBOMs, local provenance, and reproducibility verify; hosted attestation/signatures remain absent |
| Documentation complete | partial | beta site and source audit exist; user-feedback validation is unavailable |
| Public API freeze approved | blocked | inventory exists; no compatibility window or approval |
| Security reporting works | pass | SECURITY.md and GitHub private vulnerability reporting are verified |
| Rollback/hotfix procedure works | pass locally | documented and script-gated; public rehearsal pending |
| Known limitations published | pass | `KNOWN-ISSUES.md` |
| Release dry run | pass locally | clean snapshot/reproduction evidence; final rerun required |
| Public prerelease evidence sufficient | blocked | the first beta exists, but no meaningful compatibility, migration, or feedback period has elapsed |
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

1. collect a real beta evidence period and resolve verified P0/P1 reports;
2. expand supported Go, integration, adapter, and editor compatibility evidence;
3. test migration from every published prerelease to a future v1 candidate;
4. approve the public API freeze and signing/provenance policy;
5. repeat all final gates and record maintainer approval.

There is no release commit, tag, GitHub stable release, stable documentation, or
stable package-manager publication associated with this decision.
