# Stable v1.0.0 Release Decision

## Decision

**APPROVED — publish `v1.0.0`.**

The maintainer and repository owner approves the stable `v1.0.0` release with
the accepted risks recorded below. The `v0.1.0-beta.1`, `v0.1.0-beta.2`, and
`v0.1.0-beta.3` prereleases remain immutable historical evidence and are not
withdrawn.

This decision is auditable: the machine-readable gate report is
`release/gates-v1.0.0.json`, and `TestStableDecisionEvidenceIsCompleteAndHonest`
fails if a gate claims readiness without evidence or records an accepted risk
without an exception and a remediation plan.

## Exit criteria

| Criterion | Status | Evidence |
| --- | --- | --- |
| No unresolved P0/P1 | pass | Windows normalization, dependency configuration, and VS Code command-registration defects fixed; no open P0/P1 |
| Transformations differential-tested | pass | deterministic scalar-versus-batch suite in `internal/assurance` |
| Mandatory mutations killed | pass | 12/12 modeled critical mutations |
| Compatibility matrix | accepted risk | hosted Compatibility policy, Go 1.26.0/1.26.5, native and cross-target Linux/macOS/Windows, all build modes, and hermetic client checks succeeded at the tagged commit; live-backend acceptance remains out of scope |
| Upgrade from published prereleases | pass | executable migration suite from beta.1/2/3 to the v1.0.0 candidate |
| Installation | accepted risk | beta.3 public installation verified; v1.0.0 proxy behaviour observable only after publication |
| Race, fuzz, and security suites | pass | `go test -race ./...`, 18 fuzz targets, bounded soak/leak/fault, govulncheck, CodeQL, secret scan, Dependency Review |
| Artifacts verifiable | accepted risk | archives, checksums, SBOMs, local provenance, and reproducibility verify; artifacts are unsigned and carry no hosted attestation |
| Documentation complete | pass | 293 documents, 82 ADRs, published portal, 13 compile-tested examples |
| Public API freeze approved | pass | [api-freeze.md](api-freeze.md) with stable and experimental tiers |
| Security reporting works | pass | `SECURITY.md` and GitHub private vulnerability reporting |
| Rollback/hotfix procedure works | pass | documented, script-gated, and asserted by the migration suite |
| Known limitations published | pass | `KNOWN-ISSUES.md` |
| Release dry run | pass | clean-checkout reproduction at the release commit |
| Governance approval | pass | this document |

## Accepted risks

These are the exact gaps this release ships with. They are not claimed as
passing anywhere in the documentation.

1. **Unsigned artifacts, no hosted attestation.** Release automation is not
   authenticated from the release workstation, so keyless signing and build
   attestation could not be produced. Integrity is verifiable through SHA-256
   checksums, SBOMs, a local provenance statement, and reproducible builds.
   *Remediation:* enable hosted signing and attestation and publish signed
   artifacts in the next patch release.

2. **No extended production-campaign evidence at the tagged commit.** The hosted
   compatibility matrix, build-mode, cross-target, CodeQL, dependency-review, and
   release-assurance checks all succeeded at the tagged commit, but the scheduled
   long-running campaign (extended fuzz, soak, leak, and fault phases) had not
   run there; those phases were verified by their bounded local equivalents.
   *Remediation:* dispatch the production-campaign workflow and attach its
   retained artifact.

3. **Short public prerelease period.** The three betas were published on the
   same day as this release. There has been no extended external feedback
   period, so field evidence is limited to the project's own automated suites.
   *Remediation:* triage public reports promptly and ship patch releases; do not
   claim long-term production stability until hosted campaign evidence exists.

4. **No live-backend acceptance.** Client integrations are covered by hermetic
   fakes (pgxmock, miniredis, bufconn, the public gqlgen extension API), not by
   a live PostgreSQL or Redis Cluster acceptance run.
   *Remediation:* add live-backend acceptance evidence in a follow-up release.

5. **Artifact schemas remain `v1alpha1`.** Compiler and runtime artifact formats
   are not frozen; they are regenerated rather than migrated and are excluded
   from the `v1` Go API promise.

## What v1.0.0 does and does not claim

`v1.0.0` claims a frozen, Semantic-Versioned public Go API (Tier 1 in the API
freeze), a documented and enforced safety model, verified reproducible builds,
and a tested upgrade path from every published prerelease.

It does not claim universal batching, guaranteed performance improvement,
long-term production-stability evidence, signed artifacts, or live-backend
acceptance.

## Rollback

Default source mutation and active adaptive tuning remain off. Users can retain
scalar execution, use overlays, revert materialization through the backup
manifest, invalidate incompatible caches, and install a patched version. Scalar
rollback is asserted by the migration suite. See
[rollback](../rollback.md) and
[upgrade/downgrade/uninstall](../upgrade-downgrade-uninstall.md).
