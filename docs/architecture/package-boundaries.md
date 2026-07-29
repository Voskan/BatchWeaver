# Package Boundaries

BatchWeaver distinguishes between a small, stable **public** API and a larger set
of **internal** implementation packages.

## Public packages

Public packages live at the module root and are part of BatchWeaver's supported
surface. They must have a clear, long-term purpose and a deliberately small API.

| Package | Purpose | Status |
| ------- | ------- | ------ |
| `config` | User-facing configuration schema and loading | Reserved; schema version constant only |
| `diagnostics` | Dependency-free diagnostic data model | Foundational types implemented |
| `runtime` | Batching-aware execution support used by generated code | Reserved; documentation only |

Rules for public packages:

- They must not expose unstable compiler internals (AST, SSA, IR types).
- Adding to the public API requires a clear user need, tests, documentation, and
  a compatibility review.
- Generated code will eventually depend only on a minimal, stable subset of
  `runtime`.

## Internal packages

Everything under `internal/` is an implementation detail with no compatibility
guarantee. Unstable compiler and analysis logic belongs here so that it can
evolve freely without breaking users.

| Package | Purpose |
| ------- | ------- |
| `internal/buildinfo` | Build and platform identification |
| `internal/cli` | Standard-library command framework |
| `internal/filesystem` | Minimal filesystem abstraction for path resolution |
| `internal/project` | Repository root and project path discovery |

Future analysis, IR, transformation, and verification packages will also live
under `internal/` until any part of them is intentionally promoted to the public
API through review.

## Why unstable code stays internal

Go's `internal/` mechanism prevents external import, which lets the compiler
implementation change shape as the design matures without imposing breaking
changes on users. Promoting a package out of `internal/` is a deliberate,
reviewed decision, recorded in an ADR when significant.

## Naming

Package names are short, lowercase, and meaningful. Generic utility buckets such
as `util`, `common`, `helpers`, or `misc` are not used, because they attract
unrelated code and blur boundaries.
