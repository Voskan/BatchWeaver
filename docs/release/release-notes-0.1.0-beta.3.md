# BatchWeaver 0.1.0-beta.3 Release Notes

Status: public beta prerelease. This version supersedes beta.1 and beta.2 for
new installations; it is not a stable-v1 or production-readiness claim.

## Why beta.3

Public verification of a complete beta.2 GitHub Release download found that
GitHub flattened five `reports/...` asset paths while `SHA256SUMS` retained the
directory prefixes. The files and their individual GitHub digests were intact,
but whole-set verification failed. Published versions are immutable, so beta.2
was not retagged and its assets were not replaced.

Beta.3 emits every manifest-declared artifact at the release root and rejects
nested, empty, or duplicate public asset names before publication. This makes
GitHub asset names, manifest paths, and checksum paths identical.

## Installation

```bash
go install github.com/Voskan/BatchWeaver/cmd/batchweaver@v0.1.0-beta.3
go get github.com/Voskan/BatchWeaver@v0.1.0-beta.3
batchweaver version
batchweaver doctor
```

The version command must report `BatchWeaver 0.1.0-beta.3` and channel `beta`.
Install the VSIX with:

```bash
code --install-extension batchweaver-vscode-0.1.0-beta.3.vsix
```

## Verify a complete release download

Download every GitHub Release asset into one directory, then run:

```bash
shasum -a 256 -c SHA256SUMS
batchweaver release verify release-manifest.json
```

GNU systems may use `sha256sum -c SHA256SUMS`. Both commands must verify the
entire declared set, including reports, SBOMs, provenance, archives, and VSIX.

## Compatibility and safety

The supported API, schemas, ABI, feature surface, and compatibility matrix are
unchanged from beta.2. Go 1.26.5 is the supported toolchain. Preview
transformations and run the full application test suite before any explicit
materialization.

## Verification limitations

The tag and artifacts are unsigned. `SHA256SUMS` provides integrity
verification, SBOMs are supplied in SPDX and CycloneDX formats, and the local
unsigned provenance statement does not claim a hosted SLSA level.

## Security and feedback

Report vulnerabilities through the private process in `SECURITY.md`. Use the
structured issue forms or Discussions for reproducible beta feedback. No source
or runtime data is uploaded and no telemetry is added.
