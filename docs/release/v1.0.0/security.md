# Stable Security Posture Report

Local vet, lint, vulnerability, secret, license, workflow-permission, artifact,
SBOM, and support-bundle redaction checks pass at the released commit. Hosted
CodeQL and Dependency Review also pass at that commit.
Secret scanning, push protection, private vulnerability reporting, protected
main, and required release checks are enabled.

Stable security approval is granted with one supply-chain gap recorded as an
accepted risk: no hosted attestation or authenticated signing identity has been
demonstrated, so release artifacts are verifiable by checksum, SBOM, local
provenance, and reproducible rebuild but not by signature. The extended
fuzz/leak/fault campaign is dispatched against the released commit; a single
campaign is not a long-term stability claim. See `BW-KI-012` and `BW-KI-013`.

No private advisory content was available to this audit, and absence of such
content is not evidence that no vulnerability exists.

Outcome: **blocked for v1**.
