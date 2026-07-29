# ADR 0001: Project architecture

- Status: Accepted
- Date: 2026-07-29

## Context

BatchWeaver will grow into a compiler and runtime with static analysis,
transformation, generated code, a scheduler, adapters, verification, and
observability. The initial structure must support that growth without locking in
premature abstractions or exposing unstable internals as a public API.

The repository already had an `origin` remote at
`github.com/Voskan/BatchWeaver`, so the module path must match that remote.

## Decision

- Use a single root Go module, `github.com/Voskan/BatchWeaver`, matching the
  existing remote. The mixed-case path is retained because renaming the remote
  would be a destructive coordinate change and is unnecessary; Go's module proxy
  handles uppercase via escaping.
- Keep the public API intentionally small: `config`, `diagnostics`, and
  `runtime`.
- Place all unstable compiler and analysis implementation under `internal/`.
- Keep adapters in the repository initially; they may be split into separate
  modules later once their extension interfaces are stable.
- Treat standard Go toolchain compatibility as a priority throughout.

## Consequences

- Users depend only on a small, reviewed surface, and internals can evolve
  freely.
- The dependency direction is explicit and one-directional (CLI → analysis →
  transformation → runtime), which keeps the graph acyclic and testable.
- A future move of any package out of `internal/` is a deliberate, reviewed
  decision.

## Alternatives considered

- **A multi-module repository from the start.** Rejected as premature; it adds
  release and versioning overhead before the boundaries are proven.
- **Renaming the remote to an all-lowercase `batchweaver`.** Rejected because it
  changes established coordinates for no functional benefit and is not possible
  without authenticated remote access at bootstrap time.
- **Exposing analysis packages publicly for extensibility.** Rejected; the APIs
  are far from stable and would create breaking-change pressure.
