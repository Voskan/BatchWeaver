# HTTP batch endpoints

BatchWeaver posts to an explicitly declared HTTP batch endpoint with a typed JSON
envelope and maps the response back to ordered scalar outcomes. Endpoints are
never inferred from naming.

Correlation is keyed (by `request_id`, validating missing and duplicate items) or
strictly positional. The caller's `*http.Client` is used unchanged, preserving
transport, TLS, cookie jar, redirect policy, and authentication identity; callers
with different auth identities never batch. Requests are chunked deterministically
and order and duplicates are preserved.

Declare the relationship with the OpenAPI `x-batchweaver` extension or external
configuration. See [the batch extension reference](../reference/openapi-batch-extension.md).
