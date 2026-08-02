# Stabilization and Project Completion Report

## Current outcome

BatchWeaver has an implemented compiler/runtime pipeline, typed public
contracts, conservative proof and transformation stages, adapters and protocol
contracts, adaptive controls, editor tooling, and deterministic release
assurance. The public repository and first beta are published and verified;
stable v1 remains blocked pending real prerelease evidence, migration, expanded
compatibility, API-freeze approval, and final governance approval.

## Architecture and safety

Analysis, proof, transformation, runtime execution, and release publication are
separate gates. Source changes are preview-first; runtime work is scoped and
partitioned; unknown proof obligations reject; provider outcomes are validated;
and incompatible schemas, ABIs, anchors, and caches fail closed.

## Public surface

Six Go packages are importable and documented. The exported identifier baseline
is deterministic, but the v1 freeze is not approved. CLI, diagnostic, config,
bridge, artifact, daemon, editor, and generated-code compatibility surfaces are
documented independently.

## Verification evidence

Local assurance covers unit, race, differential, selected mutation, fault,
short-soak, performance budget, vulnerability, license, artifact, SBOM,
provenance, and reproducibility checks. Hosted CodeQL and Linux, macOS, and
Windows build/test jobs pass after the verified text-normalization correction.
Dependency Review passes after enabling Dependency Graph. A real Extension Host
smoke passes after fixing duplicate VS Code command registration.

## Release history

`v0.1.0-beta.1` is the first immutable public tag and GitHub prerelease.
Post-publication verification found incorrect version output on its `go install`
binary, so immutable `v0.1.0-beta.2` supersedes it with the regression fix.
Neither beta is stable-v1 evidence by itself.

## Remaining limitations

The exact list is maintained in `KNOWN-ISSUES.md`. Principal blockers are
broader compatibility, public migration evidence, API approval, hosted
attestation/signing policy, and a meaningful beta evidence period.

## Maintenance handoff

Public maintenance policy is in:

- `docs/maintainers/release-procedures.md`;
- `docs/maintainers/incident-runbook.md`;
- `docs/release/v1.0.0/stable-release-decision.md`;
- `docs/release/v1.0.0/migration.md`;
- `ROADMAP.md`.

Local coordination state remains ignored and must never be committed or included
in release artifacts.
