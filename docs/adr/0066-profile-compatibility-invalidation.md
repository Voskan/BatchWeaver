# 66. Profile compatibility and invalidation

Date: 2026-08-01

## Status

Accepted

## Context

A profile collected under a different build, ABI, config, or operation contract
may be misleading if used for active tuning.

## Decision

Profiles carry schema version, toolchain identity, runtime ABI, config digest,
and per-operation operation digests. Compatibility checks distinguish hard
incompatibility (never warm-start) from staleness (offline comparison only, or
active use with an explicit age allowance and confidence discount).

## Consequences

Stale or incompatible profiles cannot silently drive active tuning. They remain
usable for offline analysis and replay.
