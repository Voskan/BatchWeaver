# Analysis Diagnostic Reference

Static-analysis diagnostics use the BW3xxx range. Each has a stable code,
severity, message, portable location, and fingerprint. Diagnostics are sorted
deterministically by location, severity, code, and fingerprint.

## Code ranges

```text
BW3000–BW3099  package loading
BW3100–BW3199  declaration discovery and compatibility
BW3200–BW3299  symbol resolution (reserved)
BW3300–BW3399  SSA construction (reserved)
BW3400–BW3499  call graph (reserved)
BW3500–BW3599  effect summaries (reserved)
BW3600–BW3699  candidate inventory (reserved)
BW3700–BW3799  cache and reproducibility (reserved)
```

## Assigned codes

| Code | Severity | Meaning |
| ---- | -------- | ------- |
| BW3000 | error | Package loading or type-check error. |
| BW3100 | (from config) | A configuration diagnostic surfaced during discovery. |
| BW3101 | error | Conflicting declaration field across sources. |
| BW3102 | warning | Scalar symbol could not be resolved in the loaded program. |
| BW3103 | warning | Batch symbol could not be resolved in the loaded program. |
| BW3105 | error | Invalid scalar or batch symbol reference. |
| BW3108 | error | Scalar or batch function does not return an error as its last result. |

Reserved ranges are held for compatible future analysis diagnostics so that a
code is never reused for an unrelated meaning.

## Severity and exit codes

Severity never hides incomplete analysis. Warnings (for example unresolved
symbols) do not fail the build by default; use `--fail-on warning` to escalate.
See the [scan guide](../guides/scan.md) for exit codes.
