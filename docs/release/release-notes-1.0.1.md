# BatchWeaver v1.0.1

A patch release that fixes the VS Code extension source build. The Go module,
CLI, and public API are unchanged from `v1.0.0`.

## Fixed

- **`BW-KI-014` — the extension failed a clean source build.** In `v1.0.0`,
  `editors/vscode/package.json` declared version `1.0.0` while its
  `package-lock.json` still recorded `0.1.0-beta.3`. `npm ci` validates that the
  two agree, so a clean install failed and the extension could not be built or
  packaged from the `v1.0.0` source tree. The lockfile is now synchronized.

- **The extension version check no longer drifts.** The extension test read a
  hard-coded version string, so every release bump broke it after the fact. It
  now derives the expected version from `release/VERSION`, the single canonical
  version for the repository, and asserts that `package.json`, the lockfile, and
  the lockfile root package all agree with it.

The Go module, CLI, and prebuilt VSIX were unaffected by `BW-KI-014`; only
building the extension from source was.

## Compatibility

No API change. `v1.0.1` is a drop-in replacement for `v1.0.0`.

- **Tier 1 (stable, SemVer):** module root, `config`, `diagnostics`,
  `operation`, `runtime` — unchanged.
- **Tier 2 (experimental):** `bridge` and the four `adapters/*` packages —
  unchanged.
- **Go:** `go1.26.x`; minimum `go1.26.0`, current `go1.26.5`.

## Install

```bash
go install github.com/Voskan/BatchWeaver/cmd/batchweaver@v1.0.1
go get github.com/Voskan/BatchWeaver@v1.0.1
```

## Remaining accepted risks

Unchanged from `v1.0.0` except where noted; see the
[stable-release decision](v1.0.0/stable-release-decision.md).

1. **Artifacts are unsigned** and carry no hosted build attestation
   (`BW-KI-012`).
2. **No extended production-campaign evidence** at the tagged commit
   (`BW-KI-013`). The hosted compatibility matrix, build modes, cross-targets,
   CodeQL, dependency review, and release assurance did succeed at the `v1.0.0`
   commit.
3. **The public prerelease period was short.**
4. **No live-backend acceptance.** Client integrations use hermetic fakes.

Installation is no longer an accepted risk: `go install ...@v1.0.0` was verified
from the public module proxy.
