# Future Compiler Pipeline

> Status: future architecture target. None of the stages below are implemented.
> This document records the intended design so that the foundation does not
> preclude it.

The compiler is planned as the following ordered stages. Each stage consumes the
output of the previous one.

1. **Package discovery** — enumerate the Go packages in scope, using
   `go/packages` for accurate build and type information.
2. **Symbol indexing** — build an index of declarations, call sites, and types
   to support later analysis.
3. **Operation declaration loading** — load the typed declarations that pair
   scalar operations with their batch equivalents.
4. **SSA construction** — build a static single-assignment form of the relevant
   functions for precise data-flow analysis.
5. **Effect and independence analysis** — determine side effects and whether
   iterations are independent, which governs batching safety.
6. **Batch candidate detection** — identify call sites and loops that are
   candidates for batching.
7. **Semantic validation** — confirm that a candidate transformation preserves
   observable behavior.
8. **Optimization planning** — choose which candidates to transform and select a
   strategy for each.
9. **Typed transformation** — generate typed bindings and rewrite call sites (or
   integrate with the build) to use batch operations.
10. **Generated-code verification** — verify that generated code type-checks and
    behaves equivalently.
11. **Standard Go compilation** — hand the result to the standard Go toolchain,
    preserving compatibility.

## Design constraints these stages impose

To keep these stages achievable, the foundation preserves the ability to use
`go/packages`, `go/analysis`, SSA construction, source overlays, `-toolexec`
integration, and profile-guided optimization. See
[overview.md](overview.md) for the dependency direction and
[package-boundaries.md](package-boundaries.md) for where this code will live
(under `internal/`).
