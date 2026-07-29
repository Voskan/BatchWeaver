# Roadmap

This roadmap describes the planned BatchWeaver product in major phases. It
intentionally avoids fixed dates; phases will be delivered incrementally and
refined as the design matures. Only Phase 1 is complete.

## Phase 1 — Repository and engineering foundation ✅

Repository structure, build and test foundation, an initial standard-library
CLI (`version`, `help`), foundational packages, documentation, governance, and
quality automation. **Completed.**

## Phase 2 — Operation declaration model ✅

Typed contracts describing scalar operations and their batch equivalents, the
strict versioned configuration system that declares them, and the deterministic
diagnostic infrastructure that reports problems with them. **Completed.**

## Phase 3 — Runtime request coalescing

A runtime that coalesces independent logical requests into batches within a
defined scope, including deduplication and result distribution.

## Phase 4 — Static analysis and candidate discovery

Package loading and analysis to discover call sites that are candidates for
batching.

## Phase 5 — SSA and effect analysis

SSA-based analysis of data flow, effects, and independence to determine where
batching is safe.

## Phase 6 — Loop batching transformations

Semantically transparent transformation of scalar loops into batched execution.

## Phase 7 — Compile-time integration

Integration with the Go build process so transformations apply during a normal
build.

## Phase 8 — Verification engine

Differential verification that transformed code preserves observable behavior.

## Phase 9 — Adapter ecosystem

Adapters for common batchable backends (for example databases and network
services) built on stable extension interfaces.

## Phase 10 — Observability and adaptive scheduling

Metrics, tracing, and scheduling that adapts batch sizing and timing to observed
load.

## Phase 11 — IDE integration

Editor tooling to surface batching opportunities and diagnostics.

## Phase 12 — Security hardening

A focused security review and hardening pass across the compiler and runtime.

## Phase 13 — Public beta

A broader, versioned beta with compatibility commitments.

## Phase 14 — Stable release

A stable, documented release with defined compatibility guarantees.
