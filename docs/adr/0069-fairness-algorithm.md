# 69. Fairness algorithm

Date: 2026-08-01

## Status

Accepted

## Context

Multiple operations and tenant classes share the scheduler; one class must not
starve another, and priority and reserved capacity must be honored.

## Decision

The fair scheduler supports weighted fair queueing and deficit round robin, with
priority classes, per-class quotas, reserved shares, and starvation detection.
Fairness classes are anonymized; the runtime still partitions by exact tenant
internally, but metrics and reports expose only class labels.

## Consequences

Service shares are controllable and observable without exposing tenant
identities. Starvation is detected and can drive priority aging.
