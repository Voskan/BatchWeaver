# Static Analysis

BatchWeaver's static analysis (`internal/analysis`) loads real Go programs and
produces a deterministic, versioned inventory of potential batching structure
**without modifying source or executing analyzed code**. It never claims a
discovered call is safe to transform; unknown facts are represented explicitly.

## Phase pipeline

```text
load packages (go/packages)
→ classify packages + portable paths + identities
→ discover operations (typed declarations + configuration)
→ resolve symbols + signature compatibility
→ build SSA (go/ssa)
→ build call graph (CHA)
→ index scalar-operation call sites + structural context
→ compute conservative effect summaries (interprocedural fixed point)
→ inventory candidates
→ assemble deterministic, immutable Snapshot
```

The orchestration entry point is `analysis.Analyze(ctx, Request) (*Snapshot, error)`.
A non-nil error is returned only when packages cannot be loaded at all; ordinary
package and analysis problems are reported as diagnostics in the snapshot.

## Package boundaries

All analysis code is internal (`internal/analysis`) and depends on the Prompt 02
contracts, configuration, and diagnostics — never on the Prompt 03 runtime and
never on command packages. No `go/types` or SSA value is exposed as a public API.

## Immutable snapshot

A completed `Snapshot` is treated as immutable. It carries an explicit schema
version (`batchweaver.analysis/v1alpha1`, deliberately alpha), the tool and Go
versions, a build-context digest, portable package records, operations, call
sites, normalized call edges, effect summaries, candidates, and diagnostics. All
collections are sorted by stable identifiers, and no raw Go pointer identity is
serialized. In `--reproducible` mode the volatile timestamp is omitted, so JSON
is byte-identical across runs on the same inputs.

## Identity and portable paths

Identities are stable digests over canonical strings (package import path +
module, function receiver + name, call-site enclosing function + location).
Displayed short IDs derive from a SHA-256 digest. File paths are portable:
repository-relative under the workspace root, `std://` for the standard library,
and `mod://` for module-cache dependencies.

## Package loading and build context

Packages are loaded through `golang.org/x/tools/go/packages` under an explicit,
reported build context (GOOS, GOARCH, cgo, tags, tests). Package-loading and
type errors become `BW3000` diagnostics rather than being dropped.

## Declaration discovery

Operations are discovered from two sources today: **typed declarations** (calls
to the Prompt 02 `MustDeclare*`/`Declare*` helpers, found by AST inspection and
resolved via type information — never executed) and **configuration** (the Prompt
02 loader). Sources are merged with provenance; configuration overrides typed
declarations, and disagreements produce conflict diagnostics.

## SSA, call graph, and effects

SSA is built with `go/ssa` (generic instantiation enabled). The call graph uses
the conservative CHA algorithm, appropriate for library code without program
roots; direct calls remain exact and interface dispatch retains ambiguity.
Effect summaries record conservative direct effects (goroutine, channel, panic,
defer, global write, and reviewed standard-library categories) and propagate them
through a bounded, monotone interprocedural fixed point; unresolved dynamic calls
mark a summary incomplete rather than optimistically absent.

## Candidate inventory

Call sites are grouped per operation and enclosing function into candidates with
explicit states (`potential_loop`, `potential_fanout`, `potential_siblings`,
`direct_isolated`, `ambiguous_target`, `disabled_operation`,
`invalid_declaration`). Every candidate records that semantic safety has not yet
been proven.

## Current limitations

This foundation implements the core end-to-end pipeline. The following are
deferred to later prompts and are **not** exposed as stubs: SARIF and DOT output,
a content-addressed analysis cache, `//batchweaver:` directive discovery,
dependency metadata providers, RTA/VTA call-graph algorithms, multi-build-context
merging, deep generic-instantiation call-site resolution, and full source-range
mapping. Interface-dispatched operation calls are matched conservatively (by
method name and identical signature) and reported as ambiguous.
