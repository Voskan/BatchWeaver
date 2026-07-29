# Repository Conventions

## Layout

- `cmd/batchweaver/` — the thin executable entry point.
- `internal/` — implementation packages with no compatibility guarantee.
- `config/`, `diagnostics/`, `runtime/` — the small public API surface.
- `docs/` — architecture, concepts, development, and decision records.
- `examples/`, `integration/`, `testdata/` — reserved for future examples,
  integration material, and test fixtures.
- `tools/` — a separate module pinning development tools.

Directories are not padded with empty `.gitkeep` files; a directory that exists
carries a meaningful `README.md` or `doc.go` describing its role.

## Package design

- Keep package names short, lowercase, and meaningful; avoid generic buckets.
- Keep the dependency direction one-directional and acyclic (see
  [../architecture/overview.md](../architecture/overview.md)).
- Keep unstable compiler code under `internal/`.

## Go style

- All code is `gofmt`-formatted.
- Every exported identifier has a documentation comment beginning with its name.
- Error messages are lowercase, include operation context, and wrap causes with
  `%w`.

## Diagnostics

Diagnostic codes use the reserved format `BWxxxx`. Codes are allocated
deliberately as real diagnostics are introduced, not reserved in bulk.

## Commits

Commit messages follow [Conventional Commits](https://www.conventionalcommits.org/).

## Decisions

Significant architectural decisions are recorded as ADRs under
[../adr/](../adr/).
