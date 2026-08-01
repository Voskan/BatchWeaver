# 71. Deterministic offline replay

Date: 2026-08-01

## Status

Accepted

## Context

Operators need to compare policies and understand tuning decisions without
touching production or a real backend.

## Decision

A deterministic simulator replays recorded or synthetic events under a policy
against a backend timing model with a fake clock and no real backend calls. It
produces modeled backend calls, batch sizes, queue-delay percentiles, deadline
misses, and a cost score, all clearly labeled as estimates. Synthetic workload
generators are seeded and reproducible.

## Consequences

Policy comparison and counterfactual reporting are reproducible and side-effect
free. Simulated figures are never presented as measurements.
