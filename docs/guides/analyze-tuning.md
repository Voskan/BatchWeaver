# Analyze tuning

Analyze a profile to see the controller's recommendations without changing
anything:

```bash
batchweaver tune analyze --profile=service.bwp --objective=balanced
batchweaver tune analyze --profile=service.bwp --format=json
batchweaver tune analyze --profile=service.bwp --format=markdown --output=tuning.md
```

The report ranks operations by expected improvement and, per operation, shows the
current policy, measured behavior, the recommended policy, the modeled effect
(clearly labeled an estimate), confidence, and reasons.

Explain a single operation's decision:

```bash
batchweaver tune explain --profile=service.bwp --operation=users.get
```

Compare policies deterministically over a synthetic workload:

```bash
batchweaver tune replay
```
