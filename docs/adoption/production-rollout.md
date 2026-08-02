# Production Rollout and Rollback

- Pin an immutable beta and record artifact checksums.
- Begin scan-only, then overlay-only, then shadow verification.
- Canary one proven operation with bounded batch size, wait, and concurrency.
- Preserve tenant, auth, transaction, routing, and protocol partitions.
- Alert on semantic mismatches, proof invalidation, fallback rate, saturation,
  latency tails, cancellations, contract violations, and error distribution.
- Stop on any correctness uncertainty; disable the strategy and use scalar.
- Revert materialized changes, invalidate versioned caches/proofs, downgrade the
  tool/runtime/extension together, and publish a new immutable hotfix if needed.

Never hide or average away a transformation mismatch as a performance anomaly.
