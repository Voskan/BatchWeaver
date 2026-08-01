# Replay a workload

Replay compares scheduling policies deterministically against a backend timing
model, with a fake clock and no real backend calls.

```bash
batchweaver tune replay --operation=users.get --count=5000 --rate=8000
```

The output compares `current`, `latency`, and `throughput` policies by backend
calls, mean batch size, p95 queue delay, deadline misses, and a modeled cost
score. Every figure is a model estimate and is labeled as such.

Replay is reproducible: the synthetic workload is seeded, and the simulator is
fully deterministic for identical inputs.
