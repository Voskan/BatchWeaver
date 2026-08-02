# ADR 0020: Static analysis architecture and identities

- Status: Accepted
- Date: 2026-07-29

## Context

BatchWeaver needs a trustworthy program model on which later safety proofs and
transformation depend. It must load real Go programs, produce stable identities,
and be deterministic, without executing analyzed code or exposing unstable
tooling types.

## Decision

- All analysis lives in `internal/analysis`, depending on operation contracts,
  configuration, and diagnostics — never on the runtime or command packages.
- Packages load through `golang.org/x/tools/go/packages` under an explicit,
  reported build context.
- The result is an immutable, schema-versioned `Snapshot`
  (`batchweaver.analysis/v1alpha1`) with deterministically sorted collections and
  no serialized pointer identity.
- Identities are stable SHA-256 digests over canonical strings; file paths are
  portable (repository-relative, `std://`, `mod://`).

## Alternatives considered

- Exposing `go/types`/SSA objects publicly: rejected as unstable.
- A custom parser instead of official tooling: rejected; unnecessary and risky.

## Consequences

Deterministic, portable output; a stable basis for semantic proof.

## Security and compatibility

No analyzed code is executed; no secrets or absolute home paths appear in output.
The alpha schema version signals that the model is not yet a stable contract.
