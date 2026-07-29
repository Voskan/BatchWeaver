# Scheduling Policy

A scheduler policy declares how a future runtime should form batch waves. No
scheduler is implemented in this phase; the policy is validated configuration
only, and BatchWeaver makes no runtime performance guarantees.

## Modes

`immediate-wave` (the conservative default), `fixed-window`, `adaptive`,
`manual`, `throughput`, `latency`, and `deadline-aware`.

## Limits

A policy bounds batch size (min/max), aggregate weight, payload bytes, wait
duration, deadline margin, provider concurrency, queue items and bytes, active
partitions, and per-key waiters. It also selects a fairness mode and an overflow
behavior (`block`, `reject`, or `fallback`).

## Validation

Limits are checked for safety: positive sizes and concurrency, min not exceeding
max, non-negative durations, bounded queues, and mode-specific rules — for
example, `manual` mode must not configure an automatic wait, and `adaptive` mode
requires a positive maximum wait. Byte and weight limits reject invalid zero
semantics, and normalization avoids integer overflow.

## Related policies

Deduplication, retry, and fallback are separate declarative policies with their
own validation (for example, canonical deduplication requires a canonicalizer
symbol; enabled retry requires at least two attempts and a retryable
classification; parallel scalar fallback requires bounded positive concurrency).
