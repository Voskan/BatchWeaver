# Quality Gates

Every change must pass the local quality gate before it is committed or merged.
The gate is driven by the `Makefile` and mirrored in continuous integration.

## The gate

`make check` runs, in order:

| Step | Command | Purpose |
| ---- | ------- | ------- |
| Format check | `make fmt-check` | Fail if any Go file is not `gofmt`-formatted |
| Vet | `make vet` | Report suspicious constructs |
| Test | `make test` | Run unit tests |
| Race test | `make test-race` | Run unit tests under the race detector |
| Build | `make build` | Ensure the CLI builds |
| Lint | `make lint` | Run `golangci-lint` |
| Vulnerability scan | `make vulncheck` | Run `govulncheck` |
| Docs check | `make docs-check` | Lint Markdown and YAML |

## Tooling

Development tools are pinned in the `tools/` module (see
[../../tools/README.md](../../tools/README.md)) and invoked via `go run` so that
contributors do not need global installs. The lint, vulnerability, and docs
steps require network access on first run to fetch the pinned tool versions.

## Coverage

Coverage is generated with `make test-cover`. There is no artificially high
global threshold for the bootstrap; contributors are expected to cover
non-trivial branches meaningfully rather than to inflate percentages.
Documentation-only packages (those containing only `doc.go`) are not counted.

## Continuous integration

The same checks run in GitHub Actions on pushes to `main` and on pull requests.
Workflows use least-privilege permissions. See the workflows under
`.github/workflows/`.
