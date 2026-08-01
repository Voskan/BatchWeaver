# Adaptive configuration reference

Adaptive configuration is parsed with unknown-field rejection and validated for
units and ranges. Durations are strings such as `500us`, `2ms`, `24h`.

```yaml
runtime:
  adaptive:
    mode: shadow            # off | shadow | active
    objective: balanced     # latency | throughput | balanced | backend-cost | deadline-protection | custom-weighted
    profile:
      sampling_rate: 0.01   # [0,1]
      retention: 24h
    bounds:
      max_wait:      { minimum: 0s, maximum: 2ms }
      max_batch_size:{ minimum: 1,  maximum: 512 }
      concurrency:   { minimum: 1,  maximum: 16 }
    guardrails:
      p95_queue_delay: 500us
      timeout_rate: 0.001
      error_rate_regression: 0
      rollback_window: 2m
      added_latency_budget: 250us
    exploration:
      enabled: false
      maximum_step: 0.2     # max fractional change per decision
      canary_percent: 5
      cooldown: 5m
fairness:
  algorithm: weighted-fair  # weighted-fair | deficit-round-robin
  starvation_threshold: 100ms
  classes:
    - { class: interactive, weight: 3, priority: 1, reserved_share: 0.2 }
    - { class: batch, weight: 1 }
overload:
  queue_high_watermark: 0.8
  queue_critical_watermark: 0.95
  policy: shed-low-priority # accept | block | reject | fallback-direct | shed-low-priority | flush-early
```

Hard bounds are authoritative; the controller can never exceed them. Watermarks
must be in `[0,1]`.
