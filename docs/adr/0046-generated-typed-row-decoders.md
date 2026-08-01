# ADR 0046: Generated typed row decoders, no reflection in hot paths

- Status: Accepted
- Date: 2026-07-29

## Context

Row mapping must be typed and allocation-light.

## Decision

- The runtime SQL provider is generic over key and value types and takes a caller-supplied typed row decoder and key-array builder.
- The provider maps rows to outcomes by the leading request ordinal, preserving order, duplicates, and the declared missing outcome (sql.ErrNoRows).
- No reflection is used on the execution path.

## Consequences

Result mapping is typed, deterministic, and preserves scalar error identity.
