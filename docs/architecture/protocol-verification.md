# Protocol contract verification

Protocol adapters reuse the Prompt 08 verification harness to compare scalar and
batch behavior over hermetic fixtures
([ADR 0061](../adr/0061-protocol-verification-artifacts.md)).

For each case (unique, duplicate, missing, one, error), the harness runs the
scalar reference per key and the batch provider once, then compares values,
error presence and identity, and correlation, producing a deterministic
`VerificationContract` with a digest. HTTP verification uses `httptest`; GraphQL
and gRPC verification operate on the neutral model and policy layers. Artifacts
contain no credentials or raw payloads.

```bash
batchweaver http verify
batchweaver adapter verify
```
