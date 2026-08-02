# 74. Shared workspace daemon

Date: 2026-08-02

## Status

Accepted

## Context

CLI and LSP can duplicate expensive analysis for the same workspace.

## Decision

Provide an optional per-workspace daemon with a versioned local protocol over a
`0600` Unix-domain socket (never a network port), with discovery, health, and
lifecycle. The daemon owns a bounded, content-addressed, single-flight analysis
cache with memory LRU and persistent disk tiers. CLI and LSP clients use it when
a compatible daemon is running and fall back to uncached in-process analysis
when one is not available.

## Consequences

A single home for shared analysis state exists, with secure local-only access.
Keys cover workspace content, overlays, build context, toolchain, package
patterns, and analysis/proof/transformation schema versions. Cache corruption is
discarded and recomputed. No source, overlays, raw cache keys, or telemetry are
persisted or reported.
