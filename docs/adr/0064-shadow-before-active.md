# 64. Shadow mode precedes active mode

Date: 2026-08-01

## Status

Accepted

## Context

Applying tuning changes to production before their effect is understood is risky.

## Decision

The controller defaults to shadow mode, which computes and records
recommendations without changing any runtime setting. Active mode must be
explicitly enabled and is further gated by confidence, sample count, SLO
guardrails, hard bounds, and (when enabled) canary scope and cooldown.

## Consequences

Operators can observe what the controller would do before granting it the
ability to act. Active tuning is never the default.
