# 76. Tiered, debounced incremental analysis

Date: 2026-08-02

## Status

Accepted

## Context

Running deep proof/verification on every keystroke would make the editor
unresponsive.

## Decision

Schedule analysis in tiers (syntax, type, candidate, proof, verification) with
debouncing, and use a generation counter so a result from a superseded snapshot
is never published. Interactive requests take priority over background work.

## Consequences

The editor stays responsive; deep analysis runs after typing settles. This build
implements debouncing and generation-guarded publication; finer per-tier
scheduling is incremental.
