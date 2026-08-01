# Verify an adapter

Adapter verification compares scalar and batch execution across deterministic
cases and produces a signed-by-digest contract artifact.

```bash
batchweaver adapter verify
```

The command reports PASS/FAIL per case (unique, duplicate, missing, empty, one,
error) and a contract digest. A failure exits non-zero and is distinct from a CLI
usage error. Verification is read-only and never shadows writes; only pass
read-only operations to it.
