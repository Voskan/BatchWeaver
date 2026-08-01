# Verify protocol adapters

Protocol adapters are verified by comparing scalar and batch behavior over
hermetic fixtures, producing a deterministic contract digest.

```bash
batchweaver http verify        # HTTP batch endpoint (httptest)
batchweaver adapter verify     # database/sql (in-memory)
```

Verification compares values, error presence and identity, and correlation across
unique, duplicate, missing, one-key, and error cases. It is read-only and never
shadows writes. A failure exits non-zero and is distinct from a CLI usage error.
GraphQL and gRPC verification operate on the neutral model and policy layers;
end-to-end gqlgen/grpc-go verification is deferred with those client integrations.
