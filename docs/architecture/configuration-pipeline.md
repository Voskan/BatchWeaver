# Configuration Pipeline

Configuration loading is a deterministic, side-effect-free pipeline:

```text
discovery
→ decoding
→ include expansion
→ merge
→ defaults
→ normalization
→ semantic validation
→ catalog
→ canonical digest
```

## Stages

- **Discovery** (`internal/configload`) searches upward for
  `batchweaver.{yaml,yml,json}` when no path is given, stopping at the repository
  root when known. Multiple candidates in one directory are an ambiguity error.
- **Decoding** (`internal/configdecode`) parses YAML (via goccy's AST) and JSON
  (via a small position-tracking parser) into a uniform, position-aware node
  tree. Both reject unknown constructs, duplicate keys, multiple documents, and
  trailing content.
- **Include expansion** (`internal/configload`) resolves local includes relative
  to the including file, forbids remote URLs, enforces depth/file/byte limits,
  and detects cycles after symlink resolution.
- **Merge** (`internal/configmerge`) applies includes first in listed order and
  the including file last; operations merge by ID with duplicates rejected unless
  `replace: true` is set. Presence is inherent in the node tree.
- **Defaults** (`config/defaults.go`) are applied in one place, with conservative
  security defaults.
- **Normalization** (`config/normalize.go`) builds typed values: it parses IDs,
  symbols, durations, byte sizes, and enums, and constructs an `operation.Spec`
  per operation with source-located diagnostics.
- **Semantic validation** collects independent, actionable diagnostics rather
  than stopping at the first error.
- **Catalog** (`operation.Catalog`) holds the specs, rejects duplicate IDs, and
  lists them in lexicographic order.
- **Canonical digest** (`config/canonical.go`) is a deterministic SHA-256 over
  the semantic content, identical for equivalent YAML and JSON.

## Error contract

Invalid user configuration is reported as diagnostics in the `LoadResult`, not as
an opaque error. The `error` return is reserved for filesystem failures of the
primary file, context cancellation, and internal invariant failures. See
[ADR 0006](../adr/0006-strict-versioned-configuration.md).
