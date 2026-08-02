# Stable Reproducibility Report

The current local snapshot produces five platform archives, a source archive,
checksums, SPDX and CycloneDX SBOMs, local provenance, release notes, and a VSIX.
The declared artifacts reproduced byte-for-byte in two local clean outputs under
the fixed Go and Node toolchains.

Reproducibility is verified; signing is not. `v1.0.0` ships from an immutable
source tag with checksums, SBOMs, and a local unsigned provenance statement, but
no protected hosted builder, public provenance attestation, or authenticated
signature. That gap is an explicit accepted risk in the
[stable-release decision](stable-release-decision.md) and is tracked as
`BW-KI-012` in `KNOWN-ISSUES.md`.

Classification before publication:

- local declared artifacts: **bit-for-bit** in the tested environment;
- public stable artifacts: **not available**;
- hosted provenance/signatures: **not available**.

Outcome: **blocked for v1 publication**.
