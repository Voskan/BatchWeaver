# Architecture Overview

> Status: forward-looking. This document describes the target architecture. Only
> the CLI foundation and foundational packages exist today; the analysis,
> transformation, runtime, adapter, and verification layers are planned.

BatchWeaver is designed as a one-directional pipeline. Each stage depends only on
stages earlier than itself, which keeps the dependency graph acyclic and makes
each stage independently testable.

## Pipeline

```text
CLI
→ project discovery
→ configuration
→ package loading
→ static analysis
→ intermediate representation
→ optimization planning
→ source/build transformation
→ generated typed bindings
→ runtime scheduler
→ adapters
→ verification
→ observability
```

## Stage responsibilities

- **CLI** — parses commands and orchestrates the pipeline. It contains no
  compiler logic.
- **Project discovery** — locates the repository root and relevant paths
  (`internal/project`).
- **Configuration** — loads and validates the user-facing configuration schema
  (`config`).
- **Package loading** — loads Go packages and type information for analysis.
- **Static analysis** — finds candidate call sites for batching.
- **Intermediate representation** — a compiler-internal model of candidates and
  their context.
- **Optimization planning** — decides which candidates to transform and how.
- **Source/build transformation** — rewrites code or integrates with the build
  to apply batching.
- **Generated typed bindings** — typed glue between transformed call sites and
  batch operations.
- **Runtime scheduler** — coalesces, schedules, deduplicates, and distributes
  batched work (`runtime`).
- **Adapters** — connect batch operations to concrete backends.
- **Verification** — confirms that transformed programs preserve observable
  behavior.
- **Observability** — metrics and tracing across the runtime.

## Dependency direction

The dependency direction is strictly downstream: later stages may depend on the
contracts of earlier stages, never the reverse. In particular:

- The CLI must not own compiler logic.
- `diagnostics` must not depend on the CLI or compiler internals.
- The `runtime` package must not import compiler packages.
- The public `config` schema must not expose internal AST or SSA types.

These rules are enforced by review today and are candidates for automated
enforcement (for example an import-linter) in a later phase.
