# Semantic proof engine

The semantic proof engine (`internal/proof`) turns the structural candidate
inventory produced by the analysis stage into a rigorous, reproducible,
explainable eligibility decision. It proves only what it can justify from Go
semantics, program analysis, validated operation contracts, and explicit
assumptions. It never equates "no issue detected" with proof of equivalence, and
it never modifies source or changes application execution.

## Inputs

The engine consumes stable facts through the analysis package and never rebuilds
a second package loader, symbol table, SSA index, or call graph:

- the versioned analysis snapshot (`batchweaver.analysis/v1alpha1`): candidates,
  call sites, dispatch and target facts, loop and goroutine context, key
  dependency classification, and conservative effect summaries;
- validated operation contracts from configuration (kind, result, error, and
  partition semantics);
- explicit, scoped assumptions supplied by the caller.

The key dependency classification and the enclosing-function effect linkage are
computed once, in the analysis stage, and exposed as fields on each call site so
the proof engine can reason about them without SSA access.

## Pipeline

For each candidate, in deterministic order, the engine:

1. gathers normalized facts (targets, effects, receiver, context, key
   dependency) with no pointer identity;
2. resolves terminal declaration states (invalid declaration is proven
   ineligible; a disabled operation is deferred);
3. evaluates the closed obligation registry over the status lattice;
4. selects the applicable strategies for the candidate's structural class;
5. reduces each strategy's required obligations to a status and aggregates the
   candidate decision;
6. derives a deterministic proof ID and invalidation set;
7. records evidence, witnesses, assumptions, and limitations;
8. emits diagnostics for non-eligible candidates.

## Decisions and strategies

Every candidate receives one of five decisions — proven eligible, proven
ineligible, requires assumption, unknown, or deferred — always derived from named
obligations. Eligibility is reported per named strategy; there is no generic
`safe=true`. See [transformation strategies](../concepts/transformation-strategies.md)
and [proof obligations](../concepts/proof-obligations.md).

## Trust boundary

The engine trusts the Go language and memory model, the standard-library
synchronization semantics modeled by the analysis stage, validated operation
contracts, and the analysis facts for the selected build context. Everything else
is unknown until an explicit assumption supplies it. The data-race-free
prerequisite is recorded in every certificate that depends on shared-memory
reasoning. See [assumptions and trust](../concepts/assumptions-and-trust.md).

## Determinism and identity

All collections are sorted by a stable key, IDs are content-addressed over
canonical inputs, and `--reproducible` omits the timestamp. Proof IDs and JSON
output are byte-identical across machines and checkout paths for unchanged
inputs. See [ADR 0026](../adr/0026-deterministic-proof-identity.md).

## Boundaries

The proof engine produces certificates only. Source rewriting, build
interception, loop hoisting, runtime-call insertion, and provider generation are
reserved for later stages that consume these certificates. See
[ADR 0027](../adr/0027-certificates-separate-from-transformation.md).
