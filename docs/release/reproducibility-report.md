# Reproducibility Report

Snapshot binaries use Go 1.26.5, `CGO_ENABLED=0`, `-trimpath`, and
`-buildvcs=false`; version and full commit are injected, while build time is
`unknown`. Tar, gzip, and ZIP ordering, modes, ownership, and timestamps are
normalized to the Unix epoch. SBOM and provenance timestamps are also fixed.

`batchweaver release reproduce` rebuilds in a second isolated output directory
and compares every declared artifact by SHA-256 and size. Binary and metadata
artifacts are classified byte-reproducible only under the identical declared
toolchain and dependency graph. The source archive is byte-reproducible from an
identical Git index. Cross-toolchain and cross-runner reproducibility has not
been demonstrated and is not claimed. VSIX reproducibility is assessed
separately because npm package metadata and ZIP tooling can vary.
