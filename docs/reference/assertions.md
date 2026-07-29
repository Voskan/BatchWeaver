# Assertions

Assertions let a user or adapter declare a scoped fact that static analysis
cannot infer, such as the purity of a custom wrapper or the identity of a
transaction carried through an interface. This document describes the assertion
model and its current implementation status.

## Model

An assertion is a scoped assumption with a stable ID, a fully qualified symbol
scope, a set of declared facts, an origin, and a rationale. The declared facts
are drawn from a closed vocabulary:

- `side_effect_free_read`
- `panic_free`
- `receiver_immutable`
- `synchronization_equivalent`
- `result_fresh_per_call`

An assertion satisfies only the obligation types that reference the fact it
declares. It never satisfies unrelated obligations and never overrides a hard
Go-language fact. A workspace-wide (empty-symbol) assertion is rejected unless an
explicit unsafe mode is enabled.

## Requested assumptions

When an obligation could be satisfied only by an assumption that was not supplied,
the engine records a required assumption and surfaces it:

```bash
batchweaver assumption list ./...
```

No assumption is ever applied automatically, except the built-in data-race-free
prerequisite. Requested assumptions are reported with the count of candidates
that need them and a suggested action.

## Implementation status

The assumption model, trust rules, digest binding, and requested-assumption
reporting are implemented. Ingestion of assertions from configuration
(`proof.assertions`) and from a typed marker API is reserved for a later
increment; until then, assertions are supplied programmatically through the
engine input. The trust boundary and the requirement that assertions be narrow,
scoped, and visible in certificates apply to every ingestion path.
