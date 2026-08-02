# Prompt 13 Launch Handoff

Proposed version: `0.1.0-rc.1`; it is a recommendation, not a selected or tagged
version. The exact source commit must be filled from the final Prompt 12 commit
after all checks pass. Use `batchweaver release build --snapshot --output dist`
to create the five binary archives, source archive, SHA256SUMS, SPDX 2.3 and
CycloneDX 1.5 SBOMs, local provenance, and release manifest.

Before any launch, resolve every blocker in `KNOWN-ISSUES.md`, confirm branch
protection and required CI results in GitHub, run the release checklist from a
clean checkout, inspect VSIX contents and perform a real Extension Host smoke,
and obtain explicit maintainer authorization for each publication channel.
Nothing in Prompt 12 authorizes a tag, GitHub Release, Marketplace upload,
package-manager change, repository visibility change, or announcement.

Compatibility: `release/compatibility.json`. Reports: `docs/release/`. Artifact
policy: `release/config.yaml`. Rollback: `docs/release/rollback.md`. The final
snapshot manifest and checksums remain ignored local artifacts under `dist/` and
must be regenerated from the exact approved source commit.
