# Verify release artifacts

Every BatchWeaver release artifact can be verified without trusting this
documentation. There are three independent layers, strongest last.

| Layer | Answers | Available since |
| --- | --- | --- |
| Checksums | "are these the published bytes?" | `v0.1.0-beta.1` |
| Reproducible build | "can I rebuild these bytes myself?" | `v0.1.0-beta.1` |
| Signatures and attestation | "did this repository's release workflow produce them?" | see below |

> **Availability:** `v1.0.0` and `v1.0.1` artifacts are checksummed,
> SBOM-documented, and reproducible, but they are **not signed**
> (`BW-KI-012`). The signing workflow described here is committed and tested;
> the first signed artifacts ship with the next tag it runs against. Verify the
> checksum and reproducibility layers for the current releases.

## 1. Checksums

Download the artifacts and `SHA256SUMS` from the GitHub Release, then:

```bash
sha256sum --check SHA256SUMS
```

Every declared file must report `OK`. A missing file is a failure, not a
warning.

## 2. Reproducible build

Builds are deterministic under the declared toolchain: `-trimpath`, no VCS
stamping, `CGO_ENABLED=0`, and a fixed `SOURCE_DATE_EPOCH`.

```bash
git clone https://github.com/Voskan/BatchWeaver.git
cd BatchWeaver
git checkout v1.0.1
go run ./cmd/batchweaver release build --snapshot --output dist
go run ./cmd/batchweaver release verify dist/release-manifest.json
sha256sum dist/*
```

Compare the digests with the published `SHA256SUMS`.

## 3. Signatures and build provenance

Signing is **keyless**. There is no private key in this repository and no key
for you to trust: each signature is bound to the repository, the signing
workflow, the exact tag, and the GitHub Actions OIDC issuer. A signature that is
cryptographically valid but was produced by another repository, workflow, or
ref is rejected.

### One command

```bash
scripts/verify-release-signatures.sh dist v1.0.1
```

It verifies each artifact's detached signature and certificate, prints the
subject digest, and then checks the GitHub build-provenance attestation.

### Verifying manually

Install [cosign](https://docs.sigstore.dev/cosign/installation/), then for any
artifact:

```bash
cosign verify-blob \
  --signature batchweaver_1.0.1_linux_amd64.tar.gz.sig \
  --certificate batchweaver_1.0.1_linux_amd64.tar.gz.pem \
  --certificate-identity \
    "https://github.com/Voskan/BatchWeaver/.github/workflows/release-signing.yml@refs/tags/v1.0.1" \
  --certificate-oidc-issuer "https://token.actions.githubusercontent.com" \
  batchweaver_1.0.1_linux_amd64.tar.gz
```

Check the build provenance attestation with the GitHub CLI:

```bash
gh attestation verify batchweaver_1.0.1_linux_amd64.tar.gz --repo Voskan/BatchWeaver
```

### What each flag proves

- `--certificate-identity` pins the **repository**, the **signing workflow**,
  and the **exact tag**. Changing any one of them fails verification.
- `--certificate-oidc-issuer` pins the **issuer**, so a certificate from another
  identity provider is rejected.
- The artifact argument pins the **subject digest**: a modified byte fails.
- `gh attestation verify` independently confirms the artifact was built by a
  workflow in this repository.

### Coverage

Signatures and attestations cover platform archives, the VSIX, `SHA256SUMS`,
both SBOM formats, and the release manifest.

## If verification fails

Do not install the artifact. Open a
[security report](https://github.com/Voskan/BatchWeaver/security/advisories/new)
rather than a public issue, and include the artifact name, its digest, the tag,
and the exact command output.
