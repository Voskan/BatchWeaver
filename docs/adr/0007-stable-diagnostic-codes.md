# ADR 0007: Stable diagnostic codes

- Status: Accepted
- Date: 2026-07-29

## Context

Diagnostics are consumed by humans, tests, and future editor integrations, so
they need stable identifiers, deterministic ordering, and rendering that does not
depend on source tooling internals.

## Decision

- Diagnostic codes use the format `BW<CATEGORY><NNN>` (for example `BWCFG021`)
  and are validated. Reserved category ranges are documented in
  docs/reference/diagnostic-codes.md.
- Once a code is committed and documented, it keeps its meaning.
- The `diagnostics` package is dependency-free: it does not import `go/token`,
  AST nodes, SSA values, the YAML library's tokens, or the CLI.
- A `Collection` provides deterministic sorting (file, line, column, severity,
  code, message) and semantic deduplication.
- Text output is compiler-style, deterministic, and color-free by default. JSON
  output has its own schema version, stable field order, and a trailing newline.
- Both formatters are covered by golden tests.

## Consequences

- Editors and tests can rely on stable codes and deterministic output.
- The diagnostic model can be reused by every layer without import cycles.

## Alternatives considered

- **Free-form messages without codes.** Rejected; codes are needed for
  programmatic handling and documentation.
- **Coupling positions to `go/token`.** Rejected; it would leak tooling types
  into the public model.
