# Development tools

This directory pins the versions of the development tools BatchWeaver uses, so
that every contributor and CI run uses the same tool versions.

## Approach

Tools are pinned in a **separate Go module** (`tools/go.mod`) and referenced by
blank import in [`tools.go`](tools.go) under the `tools` build tag. Keeping them
in their own module ensures the main module's `go.mod` and `go.sum` stay free of
tool dependencies, which would otherwise bloat and complicate the build graph.

The `Makefile` invokes the tools with `go run <path>@<version>` using the same
pinned versions, so no global installation is required.

## Pinned tools

| Tool | Version | Purpose |
| ---- | ------- | ------- |
| `golangci-lint` | v2.12.2 | Aggregated Go linting |
| `govulncheck` | v1.6.0 | Known-vulnerability scanning |

When updating a version, change it in **both** `tools/go.mod` and the `Makefile`
so they stay in sync. Dependabot is configured to propose updates.
