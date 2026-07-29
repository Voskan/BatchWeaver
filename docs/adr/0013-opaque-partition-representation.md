# ADR 0013: Opaque partition representation and privacy

- Status: Accepted
- Date: 2026-07-29

## Context

Partitions carry potentially sensitive dimensions (tenant, authorization) and
must isolate batches exactly.

## Decision

`Partition` is an opaque, immutable value built from length-delimited components,
compared exactly over its encoding. `String()` returns a redacted, hash-derived
token, never raw components. Authorization is a fingerprint component, never a
raw token field.

## Alternatives considered

- Concatenated string keys: rejected; ambiguous grouping and leaks raw values.

## Consequences

Distinct component groupings never collide; partition tokens are safe to log.

## Security and concurrency

Raw partition data is never exposed by default; partition equality is a security
boundary enforced exactly. Values are immutable and safe to share.

## Compatibility

New API.
