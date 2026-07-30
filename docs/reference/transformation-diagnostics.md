# Transformation diagnostics

The transformation stage uses stable diagnostic codes in a sub-range chosen to
avoid collision with the analysis stage (which owns `BW3000` and `BW3100`–`BW3108`)
and the proof stage (`BW5xxx`).

## Codes

| Code | Meaning |
| --- | --- |
| `BW3401` | a transformed file does not format or parse |
| `BW3402` | a transformed package does not type-check |
| `BW3601` | an overlay path escaped workspace policy (reserved) |
| `BW3701` | a materialization precondition changed |
| `BW3702` | a revert conflict (reserved diagnostic; reported as exit code 7) |
| `BW3801` | a transformation cache entry is corrupt |

Each diagnostic carries a code, severity, message, optional location, the
candidate and plan IDs where applicable, remediation, and a stable fingerprint.

## Skip reasons

Proven candidates that are not transformed are recorded with a reason rather than a
diagnostic: `not-proven-eligible`, `unsupported-loop-form`,
`generated-binding-unavailable`, `explicit-assumption-missing`,
`overlapping-source-region`, `strategy-not-requested`, and
`source-anchor-unresolved`.

Diagnostics never claim that an automatic fix exists.
