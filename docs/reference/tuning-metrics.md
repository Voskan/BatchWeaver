# Tuning metrics reference

The adaptive layer exposes the following metrics. Labels are bounded; tenants
appear only as anonymized classes, never raw identifiers.

| Metric | Meaning |
| --- | --- |
| `batchweaver_adaptive_decisions_total` | Adaptive decisions recorded. |
| `batchweaver_adaptive_rollbacks_total` | Automatic rollbacks. |
| `batchweaver_adaptive_parameter` | Current adaptive parameter values. |
| `batchweaver_profile_samples_total` | Profile samples recorded. |
| `batchweaver_profile_dropped_total` | Samples dropped by sampling. |
| `batchweaver_slo_guardrail_breaches_total` | Guardrail breaches. |
| `batchweaver_fairness_wait_seconds` | Per-class scheduling wait. |
| `batchweaver_quota_rejections_total` | Quota rejections. |
| `batchweaver_overload_state` | Current overload state. |
| `batchweaver_load_shed_total` | Requests shed by policy. |
| `batchweaver_recursive_frontier_size` | Recursive frontier size. |
| `batchweaver_wave_duration_seconds` | Wave dispatch duration. |

Decision and wave traces carry operation, policy version, wave, adaptive mode,
decision ID, overload state, fairness class, recursive depth, and rollback state.
They never carry raw tenants or keys. Metric and exporter wiring is documented
here; concrete exporter bindings are added with the observability integration.
