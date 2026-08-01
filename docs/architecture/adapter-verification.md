# Adapter contract verification

Adapter correctness is checked by comparing scalar and batch behavior over
deterministic cases ([ADR 0050](../adr/0050-adapter-verification-artifacts.md)).

For each case (unique keys, duplicate keys, missing key, empty input, one key, and
error cases), the harness runs the scalar reference per key and the batch provider
once, then compares:

- value equality via a caller-supplied comparator;
- error presence and `errors.Is` identity (a global batch error is accepted only
  if every scalar call also failed);
- found flags and ordering.

The result is a versioned `VerificationContract` with per-case PASS/FAIL and a
deterministic digest. It contains no credentials or raw values. Verification is
read-only and never shadows writes.

Run the in-memory demonstration:

```bash
batchweaver adapter verify
```
