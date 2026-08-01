# 63. Workload profiles store no raw keys

Date: 2026-08-01

## Status

Accepted

## Context

Workload profiles must capture enough structure to tune scheduling, but must not
become a leak of operation keys, tenants, tokens, payloads, or SQL parameters.

## Decision

Profiles store only histograms, sketches, counters, and bounded categorical
counts keyed by anonymized, non-reversible class labels (keyed HMAC with a
per-session random salt in production). No raw key, tenant, token, header, URL,
or payload is ever stored. A `RawKeysStored` flag exists solely so audits can
assert it is false.

## Consequences

Profiles are safe to persist and share within a trust boundary. Cross-session
correlation of class labels is prevented by random salting; reproducible offline
collection uses a deterministic salt seed instead.
