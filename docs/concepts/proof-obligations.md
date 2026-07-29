# Proof obligations

A proof obligation is a semantic property that must hold before a transformation
strategy may be selected. Obligations have stable IDs of the form
`BW-PROOF-<FAMILY>-<NNN>` and are evaluated over a lattice rather than a boolean.

## Status lattice

- `satisfied` — the property is proven to hold.
- `violated` — the property is proven not to hold.
- `needs_assumption` — the property holds only if a named assumption is accepted.
- `unknown` — the property could not be decided within the supported precision.
- `not_applicable` — the property does not constrain this candidate or strategy.
- `deferred` — deciding the property is reserved for a later stage.

Within a strategy, a violation makes it ineligible; otherwise any unknown makes
it unknown; otherwise any needed-but-unapplied assumption makes it
assumption-required; otherwise any deferred obligation defers it; otherwise it is
eligible.

## Families and identifiers

| ID | Family | Property |
| --- | --- | --- |
| `BW-PROOF-DECL-001` | declaration | scalar and batch signatures are compatible |
| `BW-PROOF-DECL-002` | declaration | operation is enabled for transformation |
| `BW-PROOF-DECL-003` | declaration | result contract is sufficient for reconstruction |
| `BW-PROOF-OP-001` | effect | operation category permits the strategy |
| `BW-PROOF-OP-002` | effect | operation is not freshness-sensitive for reordering |
| `BW-PROOF-TARGET-001` | dependency | call target resolves to a single implementation |
| `BW-PROOF-ORDER-001` | order | no observable barrier occurs between scalar calls |
| `BW-PROOF-ORDER-002` | order | early control flow can be reconstructed |
| `BW-PROOF-DEP-001` | dependency | no key depends on a prior scalar result |
| `BW-PROOF-DEP-002` | dependency | no loop-carried dependency crosses the result |
| `BW-PROOF-RECV-001` | receiver | receiver identity is invariant across the region |
| `BW-PROOF-CTX-001` | context | context expression is invariant across the region |
| `BW-PROOF-PART-001` | transaction | receiver/tenant/transaction partitions are invariant |
| `BW-PROOF-RESULT-001` | result | duplicates and missing results reconstruct outcomes |
| `BW-PROOF-ERROR-001` | error | source-order first error can be reconstructed |
| `BW-PROOF-PANIC-001` | panic | no defer registration timing would move |
| `BW-PROOF-PANIC-002` | panic | no recover boundary is crossed |
| `BW-PROOF-CONC-001` | concurrency | strategy introduces no unsupported concurrency |
| `BW-PROOF-CONC-002` | concurrency | existing fan-out concurrency envelope is preserved |

An obligation ID is never reused for a different meaning. New properties receive
new IDs.

## Order and effect reasoning

Static movement obligations depend on the enclosing function's conservative
effect summary. A summary that is incomplete, or that contains an observable
effect (global write, channel, synchronization, network, filesystem, process,
time, randomness, logging, reflection, unsafe, goroutine launch, or an unresolved
call), yields `unknown` for the order obligation — the engine cannot prove such
an effect does not occur between the scalar calls. Only an observably effect-free,
complete summary satisfies the order obligation. Runtime coalescing does not
require the order obligation, so it may remain eligible where static prefetch is
unknown.

## Key dependency

The analysis stage classifies each key as `structural`, `result-dependent`,
`call-derived`, or `unknown`. A result-dependent key violates key independence
and the loop-carried obligation; a call-derived or unknown key yields `unknown`;
a structural key satisfies both. This is how a safe slice loop is distinguished
from a loop whose key is computed from the previous operation result.
