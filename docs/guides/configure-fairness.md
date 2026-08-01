# Configure fairness

Fairness shares scheduling across anonymized classes without starving any of
them.

```yaml
fairness:
  algorithm: weighted-fair   # or deficit-round-robin
  starvation_threshold: 100ms
  classes:
    - class: interactive
      weight: 3
      priority: 1
      reserved_share: 0.2
    - class: batch
      weight: 1
```

Inspect a fairness report (anonymized; raw tenant IDs are never shown):

```bash
batchweaver fairness report --operation=users.get
```

Quotas bound queued items, active items, concurrency, and payload bytes per
class; an admission that would exceed a quota is rejected with `BW8201`.
Starvation beyond the threshold is reported with `BW8202`.
