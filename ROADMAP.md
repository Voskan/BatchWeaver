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

## Phase 3 — Runtime request coalescing ✅

An explicit, typed runtime that coalesces independent logical requests into
bounded batches within a defined scope, with partition isolation, in-flight
deduplication, scope memoization, independent cancellation, caller-deadline-aware
scheduling, result validation, scalar fallback, and deterministic lifecycle
behavior. Invoked through explicit operation handles; automatic call
interception is not included. **Completed.**

## Phase 4 — Static analysis and candidate discovery ✅

Package loading (via `go/packages`), canonical identities, typed and
configuration declaration discovery, SSA construction, a conservative CHA call
graph, conservative effect summaries, scalar-operation call-site indexing with
structural context, a deterministic candidate inventory, and the `batchweaver
scan` command. Discovery only; semantic safety is not proven. **Completed.**

## Phase 5 — Semantic proof engine ✅

Strategy-specific semantic eligibility over the candidate inventory: a closed
proof-obligation registry evaluated on a status lattice; evaluation-order,
key-dependency, effect-barrier, receiver, context, partition, result, error,
panic/defer, and concurrency reasoning; explicit scoped assumptions and a
data-race-free trust boundary; deterministic, evidence-backed proof certificates
with witnesses; and the `batchweaver prove`, `candidate inspect`,
`proof inspect/explain/graph`, `assumption list`, and `strategy` commands. Proof
artifacts only — no source rewriting and no execution changes. **Completed.**

## Phase 6 — Loop batching transformations

Semantically transparent transformation of scalar loops into batched execution,
consuming Phase 5 proof certificates.

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
