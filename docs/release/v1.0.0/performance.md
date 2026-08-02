# Stable Performance Report

The beta preparation report in [`docs/release/performance-report.md`](../performance-report.md)
records bounded allocation and scale budgets under the pinned local environment.
Those checks prevent selected regressions; they do not establish universal
latency or throughput gains.

No public prerelease workload, external benchmark reproduction, supported
backend client matrix, or long production-like soak exists. Public documentation
must continue to describe batching as workload-dependent.

Outcome: **local budgets pass; stable performance claims blocked**.
