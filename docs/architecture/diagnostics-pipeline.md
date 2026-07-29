# Diagnostics Pipeline

The `diagnostics` package is the shared, dependency-free model for every finding
BatchWeaver reports.

## Model

- `Code` is a validated `BW<CATEGORY><NNN>` identifier.
- `Severity` is `info`, `warning`, or `error` (plus `unknown`), with strict
  parsing and JSON/text encoding.
- `Position` and `Range` carry one-based line/column and zero-based offset, with a
  repository- or user-relative file name. Absolute paths are avoided in default
  output.
- `Diagnostic` bundles a code, severity, message, range, details, source,
  related information, and advisory fixes.
- `Collection` accumulates diagnostics and provides deterministic sorting,
  semantic deduplication, and filtering.

## Sorting and determinism

Diagnostics sort by file, start line, start column, severity (errors first),
code, and message. This ordering is applied before rendering, so output is
independent of production order. Deduplication uses semantic identity (code,
severity, source, start location, message), not pointer identity.

## Rendering

- The text formatter produces compiler-style, color-free, deterministic output
  with indented details and related locations.
- The JSON formatter produces a versioned document with stable field order, no
  HTML escaping, and a trailing newline.

Both formatters are covered by golden tests, and both take an injectable writer.

## Future integration

The same model will back analyzer and verification diagnostics and a future
language-server integration. Because it depends on nothing else in BatchWeaver,
any layer can produce diagnostics without import cycles. See
[ADR 0007](../adr/0007-stable-diagnostic-codes.md).
