# ADR 0037: Overlay-first -toolexec integration with recursion prevention

- Status: Accepted
- Date: 2026-07-29

## Context

Transformed execution must integrate with standard Go tooling without maintaining two transformation engines.

## Decision

- The default architecture is overlay-first

## Consequences

 transformations are applied through a Go -overlay; the -toolexec driver observes tool actions and delegates faithfully.\n- The driver prevents recursive -toolexec invocation with a private environment marker and preserves each tool's exit code without a shell.\n- There is one overlay transformation engine, not a parallel tool-exec generator.:Standard go build/test/run work unchanged, recursion is prevented, and there is a single source of transformation truth.
