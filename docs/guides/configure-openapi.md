# Configure HTTP / OpenAPI

Declare a batch endpoint with the `x-batchweaver` OpenAPI extension (see
[the extension reference](../reference/openapi-batch-extension.md)) or external
configuration.

## Validate and inspect

```bash
batchweaver openapi validate --file=api.yaml
batchweaver openapi inspect  --file=api.yaml
```

## Verify

```bash
batchweaver http verify
```

`http verify` runs a hermetic in-memory batch server and the HTTP provider,
comparing scalar and batch behavior across cases and printing a contract digest.

## Transport

The caller's `*http.Client` is used unchanged, preserving TLS, cookie jar,
redirect policy, and auth identity; callers with different auth identities never
batch. Requests are chunked by item limit; order and duplicates are preserved.
