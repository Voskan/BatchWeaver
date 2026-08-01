# Enable bounded adaptive mode

Adaptive tuning is disabled by default. Shadow mode recommends; active mode
applies bounded changes subject to every safety gate.

```yaml
runtime:
  adaptive:
    mode: active
    bounds:
      max_wait:
        minimum: 0s
        maximum: 2ms
      max_batch_size:
        minimum: 1
        maximum: 512
      concurrency:
        minimum: 1
        maximum: 16
    guardrails:
      p95_queue_delay: 500us
      timeout_rate: 0.001
      error_rate_regression: 0
      rollback_window: 2m
      added_latency_budget: 250us
    exploration:
      enabled: false
      maximum_step: 0.2
      canary_percent: 5
      cooldown: 5m
```

Hard bounds are authoritative: the controller can never exceed them. Active
changes are gated on confidence, sample count, guardrails, and (when exploration
is enabled) canary scope and cooldown. An emergency freeze restores configured
defaults immediately and does not depend on the profile store.
