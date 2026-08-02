# Stable Performance Report

The beta preparation report in [`docs/release/performance-report.md`](../performance-report.md)
records bounded allocation and scale budgets under the pinned local environment.
Those checks prevent selected regressions; they do not establish universal
latency or throughput gains.

The repository now contains a scheduled production-like campaign with a
configurable long soak, concrete-client and editor phases, race detection,
resource budgets, and retained per-commit evidence. The harness has bounded
local smoke evidence; multiple successful hosted runs have not yet been
collected at a final v1 candidate commit. No external benchmark reproduction or
downstream production workload is available. Public documentation must continue
to describe batching as workload-dependent.

Outcome: **campaign infrastructure and local budgets pass; stable performance
claims blocked pending hosted final-candidate and downstream evidence**.
