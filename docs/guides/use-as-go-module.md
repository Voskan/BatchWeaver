# Use BatchWeaver as a Go Module

BatchWeaver contains importable Go packages as well as the `batchweaver` CLI.
The module path is:

```text
github.com/Voskan/BatchWeaver
```

## Install the beta

The current public prerelease is `v0.1.0-beta.3`. Install the library with:

```bash
go get github.com/Voskan/BatchWeaver@v0.1.0-beta.3
```

Install the command separately with:

```bash
go install github.com/Voskan/BatchWeaver/cmd/batchweaver@v0.1.0-beta.3
```

Then browse these package pages:

- `https://pkg.go.dev/github.com/Voskan/BatchWeaver`
- `https://pkg.go.dev/github.com/Voskan/BatchWeaver/operation`
- `https://pkg.go.dev/github.com/Voskan/BatchWeaver/runtime`
- `https://pkg.go.dev/github.com/Voskan/BatchWeaver/config`
- `https://pkg.go.dev/github.com/Voskan/BatchWeaver/diagnostics`

## Import the typed contracts

```go
import (
    batchweaver "github.com/Voskan/BatchWeaver"
    "github.com/Voskan/BatchWeaver/operation"
)
```

Use `batchweaver.MustDeclareFunction` or `MustDeclareMethod` for package-level
declarations. Use the error-returning variants when declarations are assembled
dynamically. The [README example](../../README.md#use-the-go-package) and
[`examples/declarations/basic`](../../examples/declarations/basic) are compiled
by the normal Go test suite.

## Version publication procedure

Go modules are published by immutable semantic-version Git tags, not by
uploading an archive to pkg.go.dev. The release owner must:

1. merge the reviewed release commit to the intended release branch;
2. pass all mandatory repository and release gates;
3. create and push the unique semantic-version tag;
4. verify the tag through `proxy.golang.org` with
   `GOPROXY=https://proxy.golang.org go list -m`;
5. verify package documentation and examples on pkg.go.dev;
6. never move or reuse the published tag.

The detailed runbook is in the [release checklist](../release/release-checklist.md).

## Stability

Versions below v1 are prerelease APIs. Pin an exact version, keep CLI, bridge,
runtime, and generated artifacts aligned, and regenerate proof/transformation
artifacts after upgrades. Stable compatibility begins only after the v1 API
freeze and release decision pass.
