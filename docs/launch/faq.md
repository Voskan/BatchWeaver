# Beta FAQ

## Does it batch every call?

No. Only documented patterns with satisfied proof obligations are eligible.

## Does it change source automatically?

No. Plans, diffs, overlays, and tests are the default. Materialization is
explicit and reversible.

## Is it always faster?

No. Performance depends on workload and backend. Use reproducible measurements.

## Is it production ready?

The stable release freezes the Tier 1 Go API. It still makes no universal production claim: unsupported patterns stay scalar.

## Does it upload code or telemetry?

No remote telemetry or source upload is enabled by default.

## How do I report a semantic mismatch?

Use the safety form for sanitized non-sensitive cases. Use private security
reporting for exploitable, isolation, authorization, or data-exposure cases.
