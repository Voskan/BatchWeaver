# Build and Offline Verification

Required: Go 1.26.x (minimum 1.26.0; release pin 1.26.5). VS Code packaging uses
Node 22, npm 10, and the committed lockfile. Release binaries require no native
dependency because CGO is disabled.

After dependencies are cached, checksums, manifest validation, archive-layout
validation, and provenance inspection work without network access. Module and
npm installation, vulnerability database refreshes, and external link checks
may require network access. A pre-populated Go module cache is supported; a
fully air-gapped npm build has not been tested. Go module vendor and go.work
modes are exercised by the hosted compatibility matrix. Runtime, compiler,
daemon, and editor features do not fetch remote schemas by default.
