# ADR 0002: Go version policy

- Status: Accepted
- Date: 2026-07-29

## Context

BatchWeaver relies on modern Go tooling (`go/packages`, `go/analysis`, SSA, and
toolchain integration). It needs a predictable, reasonably current Go version,
and it must build reproducibly even on machines whose installed Go differs from
the project's target.

At bootstrap, the host machine had `go1.24.2` installed, while the project
targets Go 1.26.

## Decision

- Require **Go 1.26** as the minimum (`go 1.26` in `go.mod`).
- Pin the toolchain to **`go1.26.5`** (`toolchain go1.26.5`).
- Rely on the default `GOTOOLCHAIN=auto`, which fetches the pinned toolchain
  automatically when the installed Go is older. This was verified to work before
  adopting the pin.
- Accept newer Go 1.26 **patch** releases after verification. If the locally
  installed Go 1.26 patch is newer than the pin, the pin may be advanced to that
  patch.
- Treat a move to Go 1.27 or later, or to a development toolchain, as an explicit
  compatibility change requiring its own decision.

## Consequences

- Contributors and CI build against a consistent toolchain regardless of their
  system Go, without a manual install step.
- The project stays on a stable release line and does not silently drift onto
  unreleased toolchains.

## Alternatives considered

- **Target the host's Go 1.24.** Rejected; it forgoes required Go 1.26 tooling
  behavior the project depends on.
- **Leave the toolchain unpinned.** Rejected; unpinned builds are less
  reproducible and can diverge between contributors and CI.
