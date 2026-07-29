# Explaining a rejection

When a candidate is not proven eligible, the proof engine records exactly why.

## Read the candidate

```bash
batchweaver candidate inspect <candidate-id>
```

A rejected candidate prints the failed or unknown obligation and a witness — a
concrete trace of the source, effect, or dependency that blocks the strategy — as
well as the independent runtime-coalescing eligibility.

## Common rejections

- **Write operation** — a non-idempotent write is `proven_ineligible` for every
  strategy because the core engine does not batch writes. Reason code
  `write_category`.
- **Ambiguous target** — an interface call whose implementation is not resolved
  is `unknown`, including for runtime coalescing, because the engine never merges
  unresolved targets. Reason code `ambiguous_target`.
- **Loop-carried key** — a key computed from a previous operation result violates
  key independence, so static prefetch is `proven_ineligible`; runtime coalescing
  may still apply.
- **Observable barrier** — an observable effect in the enclosing function makes
  the order obligation `unknown`, so static prefetch is `unknown`; runtime
  coalescing may still apply. Reason code `observable_barrier`.
- **Missing contract** — without a validated result and partition contract, the
  reconstruction and partition obligations are `unknown`.

## Unknown versus ineligible

`unknown` is a precision result: a future, more precise analysis or an explicit
assumption could change it. `proven_ineligible` is a definitive semantic conflict.
Neither is a defect in your code; both prevent an unsafe transformation.
