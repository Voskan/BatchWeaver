# Beta Operations Runbook

## Daily during launch week

Verify the repository, prerelease flag, asset list/digests, docs URL, security
link, and installation smoke. Triage new safety/security reports first. Record
only observed issues and feedback in the launch-health report.

## Weekly

Review CI, dependency alerts, compatibility reports, performance evidence,
documentation gaps, stale proofs/schemas, and supported-version messaging. Do
not close safety or security reports through stale automation.

## Severity and hotfix

- P0: data/security/isolation/correctness; disable affected behavior and begin
  private incident handling immediately.
- P1: a major supported workflow is broken; prioritize a new prerelease.
- P2: important defect with a documented workaround.
- P3: minor defect.

Never replace a tag or asset. Mark an unsafe release clearly, retain evidence,
publish an advisory when appropriate, and issue a higher immutable version.

## Feedback classification

Classify each real item as correctness, compatibility, usability, performance,
documentation, adapter request, editor, false positive, false negative, or
unsupported use case. Absence of feedback is not stability evidence. Stable
release review uses `docs/release/beta-exit-criteria.md`.
