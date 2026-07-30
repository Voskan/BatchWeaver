# ADR 0030: Go command overlays as the default non-mutating path

- Status: Accepted
- Date: 2026-07-29

## Context

Developers must be able to build, test, and run transformed code without editing
their source tree, and the analyzed, type-checked, and compiled bytes must be
identical.

## Decision

- Transformed execution is non-mutating by default. Transformed files are written
  to a content-addressed cache under the ignored `.batchweaver/` state directory
  and exposed through a standard Go `-overlay` manifest.
- The same transformed bytes back in-process package loading (via
  `packages.Config.Overlay`) and the Go command (`-overlay`), so analysis, type
  checking, and compilation observe one consistent view.
- `batchweaver build`, `test`, and `run` delegate to the installed Go tool with
  an explicit argument array and no shell, preserving its exit code.

## Consequences

The working tree is never modified by build/test/run. Source mutation requires the
separate, explicit `materialize` command.
