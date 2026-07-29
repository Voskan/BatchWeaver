# ADR 0026: Deterministic proof identity and invalidation

- Status: Accepted
- Date: 2026-07-29

## Context

Proof certificates must be cacheable and comparable across machines and repeated
runs, and they must become invalid when any input that could change the decision
changes.

## Decision

- A proof ID derives from the canonical candidate digest, the ordered obligation
  outcomes, the strategy outcomes, the contract digest, the assumption digest,
  and both the proof schema and strategy-registry versions.
- Volatile inputs (timestamps, host paths, pointer identities) never contribute
  to identity. `--reproducible` omits the timestamp for byte-identical output.
- Each certificate carries an invalidation set naming the analysis, contract,
  assumption, and candidate digests plus the schema and registry versions.
- Source references use the portable locations already normalized by the
  analysis package, so IDs are identical across checkout paths.

## Consequences

Identical inputs yield identical certificates; any change to source, contracts,
assumptions, the analyzer, or the proof schema invalidates affected certificates.
