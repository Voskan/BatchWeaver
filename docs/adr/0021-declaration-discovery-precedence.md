# ADR 0021: Declaration discovery and precedence

- Status: Accepted
- Date: 2026-07-29

## Context

Operations can be declared through Prompt 02 typed declarations and
configuration. Discovery must unify them deterministically, statically, and
without executing user code.

## Decision

- Typed declarations are discovered by inspecting the AST for calls to the Prompt
  02 `Declare*`/`MustDeclare*` helpers, resolving operands via type information.
- Configuration declarations reuse the Prompt 02 loader; no second parser is
  created.
- Sources merge with provenance. Configuration overrides typed declarations, and
  disagreements produce conflict diagnostics rather than silent merges.
- Symbols are resolved to `types.Func` objects; compatibility is conservative
  (valid, unresolved, or invalid).

## Alternatives considered

- Executing declarations to read runtime values: rejected; discovery must be
  static.
- Silent precedence without diagnostics: rejected; conflicts must be visible.

## Consequences

Every operation retains its declaration sources; conflicts are reported.

## Security

No user code runs during discovery. Directive and dependency-metadata sources are
deferred and documented, not stubbed.
