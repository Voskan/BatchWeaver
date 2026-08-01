# Handle overload

Overload control protects the scheduler under load.

```yaml
overload:
  queue_high_watermark: 0.8
  queue_critical_watermark: 0.95
  policy: shed-low-priority   # accept | block | reject | fallback-direct | shed-low-priority | flush-early
```

Inspect the state and admission decisions for a set of signals:

```bash
batchweaver overload inspect --queue=0.97 --timeout-rate=0.0 --policy=shed-low-priority
```

Requests are never shed silently: a shed or rejected request carries a typed
diagnostic (`BW8302`, `BW8303`), critical requests are protected, and a
backpressure signal is exposed to callers and adapters.
