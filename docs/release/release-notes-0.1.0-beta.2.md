# BatchWeaver 0.1.0-beta.2 Release Notes

Status: public beta prerelease. This version supersedes beta.1 for new
installations; neither release is a stable-v1 or production-readiness claim.

## Why beta.2

Public verification of `go install ...@v0.1.0-beta.1` found that the installed
CLI reported `dev`. The beta.1 module, packages, and release archives were
otherwise intact. Semantic-versioned releases are immutable, so beta.1 was not
retagged or replaced.

Beta.2 derives its version and available revision from Go module build
information whenever release linker metadata is absent. It also includes the
corrected GitHub Pages workflow pin discovered during the beta.1 deployment.

## Installation

```bash
go install github.com/Voskan/BatchWeaver/cmd/batchweaver@v0.1.0-beta.2
go get github.com/Voskan/BatchWeaver@v0.1.0-beta.2
batchweaver version
batchweaver doctor
```

The version command must report `BatchWeaver 0.1.0-beta.2` and channel `beta`.

Archives must be checked against `SHA256SUMS`. Install the VSIX with:

```bash
code --install-extension batchweaver-vscode-0.1.0-beta.2.vsix
```

## Compatibility and safety

The supported feature surface and compatibility matrix are unchanged from
beta.1. Go 1.26.5 is the supported toolchain. Concrete pgx, go-redis, gqlgen,
and grpc-go bindings are not included. Preview transformations and run the full
application test suite before any explicit materialization.

## Verification limitations

The tag and artifacts are unsigned. `SHA256SUMS` provides integrity
verification, SBOMs are supplied in SPDX and CycloneDX formats, and the local
unsigned provenance statement does not claim a hosted SLSA level.

## Security and feedback

Report vulnerabilities through the private process in `SECURITY.md`. Use the
structured issue forms or Discussions for reproducible beta feedback. No source
or runtime data is uploaded and no telemetry is added.
