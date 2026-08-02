# Compiler Pipeline

> This path is retained for historical links. The pipeline described here is
> implemented; current architecture is summarized in [overview.md](overview.md).

The compiler executes ordered, independently testable stages:

1. package discovery and loading with `go/packages`;
2. typed declaration and symbol indexing;
3. SSA and conservative call-graph construction;
4. effect summaries and structural candidate discovery;
5. strategy-specific semantic proof obligations;
6. deterministic proof certificates with source evidence;
7. versioned transformation planning and source anchors;
8. typed bridge generation and source/build overlays;
9. type checking, transformed build/test/run, and source maps;
10. optional atomic materialization, backup, recovery, and revert;
11. execution through the typed runtime and explicit providers.

Unsupported shapes, incomplete call resolution, failed obligations, stale source
anchors, or schema/ABI mismatches stop the pipeline with diagnostics. They do
not produce partial rewrites.

Compiler implementation remains under `internal/`. Generated code depends on
the public `bridge`, root contracts, `operation`, and `runtime` packages rather
than AST, SSA, or transformation implementation types.
