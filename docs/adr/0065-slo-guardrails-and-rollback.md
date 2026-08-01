# 65. SLO guardrails and automatic rollback

Date: 2026-08-01

## Status

Accepted

## Context

Even a bounded change can regress latency, timeouts, or error rate under real
traffic.

## Decision

Every active change is subject to SLO guardrails. Before application, a modeled
added-latency budget must be satisfied. After application, a rollback monitor
watches measured metrics for a configured window and restores the prior settings
if a guardrail is breached (queue delay, timeout rate, error-rate regression,
throttling, fairness regression, verification mismatch, or contract violation).

## Consequences

A bad change is self-correcting within the rollback window. Guardrails are
authoritative over the controller's objective.
