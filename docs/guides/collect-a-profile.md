# Collect a workload profile

A profile captures privacy-safe workload statistics for tuning.

```bash
batchweaver profile collect \
  --output=.batchweaver/profiles/service.bwp \
  --count=5000 --rate=8000
```

This build collects from a deterministic synthetic workload; in production the
collector is fed by the live runtime. The command reports operations observed,
logical and backend calls, and redacted fields, and confirms that no raw
operation keys were stored.

Inspect, merge, validate, and check redaction:

```bash
batchweaver profile inspect  --profile=service.bwp
batchweaver profile merge    a.bwp b.bwp --output=merged.bwp
batchweaver profile validate --profile=service.bwp --abi=batchweaver.bridge/v1alpha1 --max-age=24h
batchweaver profile redact   --profile=service.bwp
```

`validate` distinguishes hard incompatibility from staleness; a stale profile
may still be used for offline comparison.
