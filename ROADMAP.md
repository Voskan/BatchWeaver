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

## Phase 6 — Loop batching transformations ✅

Versioned transformation IR consuming Phase 5 proof certificates; certificate
validation and stable source anchoring; deterministic rewrite planning; the first
production strategy (static slice/array loop prefetch for certified read-only
operations) with typed key collection, a single batch call, global-result
validation, and source-order replay; build overlays and transformed
`build`/`test`/`run`; deterministic unified diffs and source maps; and atomic
materialization with backup, revert, and recovery. Certified read-only slice/array
loops only; other candidate classes remain discovered and proven but not yet
rewritten. **Completed.**

## Phase 7 — Runtime call lowering and fan-out coalescing ✅

Typed, reflection-free runtime bridge ABI (`bridge` package); lowering of
certified scalar calls into `bridge.Operation.Call` for standalone, sibling, and
existing goroutine/errgroup fan-out call sites, coalescing through the Prompt 03
runtime with a guaranteed scalar fallback; context/cancellation/deadline/partition
and error preservation; explicit batching barriers; an overlay-first `-toolexec`
driver with recursion prevention and `tool-exec doctor`/`explain`; and
`runtime inspect`/`barrier inspect`. Runtime lowering reuses the Phase 6 IR,
overlays, and materialization. No backend batch synthesis. **Completed.**

Prompt 08 targets the provider/adapter SDK and the first production backend
adapters (`database/sql`/pgx, Redis), covered by the adapter phase below.

## Phase 8 — Verification engine

Differential verification that transformed code preserves observable behavior.

## Phase 9 — Adapter ecosystem 🚧

Adapters for common batchable backends built on a stable extension interface.
Delivered: the versioned adapter SDK and manifest/capability model; a narrow real
SQL parser with exact-key PostgreSQL read synthesis; a typed reflection-free
`database/sql` batch provider; Redis cluster hash-slot grouping and MGET/HMGET/
pipeline mapping; and scalar/batch contract verification. Concrete pgx and
go-redis client bindings are contract-defined but deferred (offline dependency).

Prompt 09 delivered the network protocol layer: a fully implemented HTTP explicit
batch adapter (net/http + typed JSON envelopes + OpenAPI x-batchweaver binding),
GraphQL resolver-wave analysis with a real query parser and one-scope-per-operation
semantics, an explicit gRPC batch-binding and metadata policy layer, and protocol
contract verification. Concrete gqlgen and grpc-go client integrations are
contract-defined but deferred (offline dependency). Prompt 10 targets adaptive
scheduling and cost-based tuning.

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
