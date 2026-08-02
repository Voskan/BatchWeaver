# ADR 0008: Declarations without global registration

- Status: Accepted
- Date: 2026-07-29

## Context

Operations must be declared in user code in a way that the analyzer can
discover statically and the runtime can consume, without hidden
initialization-time side effects.

## Decision

- Typed declarations are plain, immutable-by-convention package-level values
  created by `DeclareFunction`/`MethodDeclaration` (and their `Must` variants).
- There is no package-level mutable registry and no `init`-time registration.
- The canonical declaration shape is a package-level `var` assigned from
  `MustDeclareMethod`/`MustDeclareFunction`, which the AST analyzer can find
  reliably.
- `Must` constructors panic only for programmer errors during initialization,
  with deterministic messages that include the operation ID and no pointer
  addresses.

## Consequences

- Declarations are statically discoverable and free of import-order surprises.
- The runtime consumes generated registries without any hidden global
  state introduced here.

## Alternatives considered

- **An `init`-time global registry.** Rejected; it creates hidden ordering
  dependencies and is hard to analyze statically.
- **Reflection-based discovery at runtime.** Rejected; static discoverability is
  a core design goal and reflection is avoided on hot paths.
