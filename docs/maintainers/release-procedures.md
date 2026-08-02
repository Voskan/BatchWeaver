# Release Procedures

## Version policy

- Patch releases contain compatible fixes, security hardening, and severe
  compatibility or installation corrections.
- Minor releases add compatible features and require API, documentation,
  migration, benchmark, and compatibility review.
- Major releases may change stable public contracts and require explicit
  migration tooling and governance approval.
- Experimental APIs are exempt only when their package, schema, or documentation
  explicitly marks them experimental.

## Patch release

1. reproduce and classify the defect;
2. add a focused regression test;
3. implement the smallest compatible fix;
4. run affected, race, security, API, artifact, and installation gates;
5. update changelog, known issues, and release notes;
6. build from a clean pinned environment;
7. create one immutable tag and release;
8. verify public assets, proxy/module installation, docs, and rollback.

## Minor release

A minor release additionally requires roadmap alignment, an ADR for
cross-cutting design, public API review, migration documentation, compatibility
matrix updates, and reproducible performance evidence for public claims.

## Security release

Use a private advisory and fix branch. Limit access, reproduce privately, assess
affected versions, prepare the patch and advisory, build release artifacts, and
coordinate disclosure only after users can obtain the fixed version. Never copy
private reports into public issues.

## Backports

Backport security, semantic correctness, data/isolation safety, severe supported
compatibility, and installation fixes. Do not backport ordinary features. Each
backport must preserve the target branch's API and pass its release gates.

## Support cadence

- security dependencies: review promptly when an advisory affects supported use;
- patch/minor dependencies: group into reviewable updates;
- major dependencies: compatibility and license review required;
- Go, gopls, VS Code, and adapter client versions: test before declaring support;
- no LTS branch or response SLA is promised without named maintainers and
  sustainable infrastructure.

## Retention and retraction

Keep release manifests, checksums, SBOMs, provenance, notes, compatibility, and
known-issue records for every published version. Never replace assets silently.
Use Go module retraction only for a published version that must no longer be
selected, and publish the retraction from a newer immutable version.
