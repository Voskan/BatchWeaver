# ADR 0009: YAML decoder selection

- Status: Accepted
- Date: 2026-07-29

## Context

YAML is part of BatchWeaver's documented configuration format, so a YAML parser
is required. It must provide source positions and support strict decoding, and
its license must be compatible with Apache-2.0 distribution.

## Decision

- Use `github.com/goccy/go-yaml`, pinned at `v1.19.2`, for YAML parsing.
- Reasons: it is actively maintained, MIT-licensed (compatible with Apache-2.0
  distribution), and exposes a position-aware AST via its `parser` and `ast`
  packages, which is essential for source-located diagnostics.
- BatchWeaver does not decode YAML directly into its schema structs. Instead, the
  parser output is converted into an internal, position-aware node tree
  (`internal/configdecode`), decoupling the public diagnostics and schema from
  the dependency's token types.
- JSON is parsed by a small, dependency-free, position-tracking parser in the
  same package, so JSON gains the same strictness (duplicate-key rejection,
  single-document, no trailing content) and positions as YAML.
- After adding the dependency, `go mod tidy`, `go mod verify`, and `govulncheck`
  were run; no vulnerabilities were reported.

## Consequences

- Configuration diagnostics carry accurate line and column numbers.
- The YAML dependency can be replaced later without changing the public API,
  because only the internal node-conversion layer depends on it.

## Alternatives considered

- **`gopkg.in/yaml.v3`.** It supports `KnownFields` but offers weaker position
  information for building rich, source-located diagnostics.
- **A hand-written YAML parser.** Rejected as unjustified scope; YAML is complex
  and a maintained library is safer.
