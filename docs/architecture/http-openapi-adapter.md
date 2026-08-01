# HTTP / OpenAPI adapter

The HTTP adapter binds explicit batch endpoints over the standard `net/http`
client and maps typed JSON envelopes back to ordered scalar outcomes. It is fully
implemented and hermetically tested with `httptest`.

## Explicit endpoints only

An HTTP batch endpoint is used only when explicitly declared — via an OpenAPI
`x-batchweaver` extension or user configuration — never inferred from endpoint
naming ([ADR 0060](../adr/0060-no-inferred-http-batching.md)). A batch endpoint
without a stable item correlation key is rejected.

## Envelopes

The `HTTPProvider` sends a typed request envelope (`items[]` of
`{request_id, key}`) and decodes a typed response envelope. Correlation is either
**keyed** (by `request_id`, validating missing/duplicate) or strictly
**positional**. Values are pointers so absent, null, and zero remain distinct.
Requests are chunked deterministically by item limit; order and duplicates are
preserved.

## Transport identity

The caller-owned `*http.Client` is used as-is, preserving its transport, TLS,
cookie jar, redirect policy, and authentication identity. Different auth
identities never batch.

## OpenAPI loading

`LoadOpenAPI` parses OpenAPI 3.1+ (JSON via the standard library, YAML via the
existing goccy dependency) with a bounded document size and no remote reference
resolution, discovering `x-batchweaver` batch bindings
([ADR 0059](../adr/0059-openapi-vendor-extension.md)).

## CLI

```bash
batchweaver openapi validate --file=api.yaml
batchweaver openapi inspect  --file=api.yaml
batchweaver http verify
```
