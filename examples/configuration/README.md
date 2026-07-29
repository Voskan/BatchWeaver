# Configuration examples

- [batchweaver.yaml](batchweaver.yaml) — a complete valid configuration in YAML.
- [batchweaver.json](batchweaver.json) — the same configuration in JSON.

Both files describe the **same semantic configuration**, so they produce an
identical semantic digest. Validate and inspect them with:

```bash
batchweaver config validate --file examples/configuration/batchweaver.yaml
batchweaver config validate --file examples/configuration/batchweaver.json
batchweaver operation list --file examples/configuration/batchweaver.yaml
batchweaver config print --file examples/configuration/batchweaver.yaml --format json
```

These examples exercise only the configuration and operation-model contracts; no
batching or scheduling is performed.
