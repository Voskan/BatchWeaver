# CI Rollout

Add evidence in phases; materialization is never required:

```text
scan report → proof report → transform verify → transformed tests
→ runtime shadow verification → performance budget
```

Pin the BatchWeaver version, Go toolchain, configuration digest, runtime ABI,
and generated-artifact schemas. Store reports as bounded CI artifacts without
source or secrets. Treat proof changes, new assumptions, unsupported candidates,
semantic mismatches, race failures, or budget regressions as review events.

Privileged workflows must not execute untrusted fork code with write tokens or
release secrets.
