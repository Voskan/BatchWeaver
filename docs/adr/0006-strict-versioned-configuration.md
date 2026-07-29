# ADR 0006: Strict versioned configuration

- Status: Accepted
- Date: 2026-07-29

## Context

Configuration controls batching semantics, so it must be unambiguous,
reproducible, and safe against surprising input. It must also support both YAML
and JSON and produce identical results from either.

## Decision

- The schema is explicitly versioned; a missing `version` is an error and an
  unsupported version is rejected.
- Decoding is strict: unknown fields, duplicate keys, multiple documents, and
  trailing content are errors. Unsafe YAML constructs (anchors, aliases, tags,
  merge keys) are rejected.
- Includes are local only (no remote URLs), resolved relative to the including
  file, bounded by depth/file/byte limits, and cycle-checked after symlink
  resolution.
- Merge semantics are explicit: includes are applied first in listed order, the
  including file last; operations merge by ID with duplicates rejected unless
  `replace: true` is set.
- Defaults are centralized in one normalization step; security-sensitive defaults
  are conservative.
- A deterministic SHA-256 semantic digest excludes source paths, positions, and
  include order, so equivalent YAML and JSON yield the same digest.
- Configuration is decoded through an internal position-aware node tree, keeping
  the public diagnostics independent of the YAML dependency's token types.

## Consequences

- Misconfiguration is caught early with precise, source-located diagnostics.
- The digest gives a stable identity for a configuration across formats.

## Alternatives considered

- **Lenient decoding that ignores unknown fields.** Rejected; it hides typos and
  drift.
- **Assuming a default schema version.** Rejected; explicit versioning is safer
  for future migrations.
