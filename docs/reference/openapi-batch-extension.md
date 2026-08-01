# OpenAPI x-batchweaver extension

The versioned `x-batchweaver` OpenAPI vendor extension declares a scalar/batch
relationship on a batch operation.

```yaml
paths:
  /users:batchGet:
    post:
      operationId: batchGetUsers
      x-batchweaver:
        scalar-operation-id: users.get
        mode: keyed            # keyed | positional
        request-items-path: /items
        request-key-path: /key
        response-items-path: /items
        response-key-path: /request_id   # required for keyed mode
        per-item-error-path: /error
        maximum-items: 500
```

Rules:

- `scalar-operation-id` is required.
- `keyed` mode requires `response-key-path`; `positional` mode requires the
  response to contain exactly one item per request item in order.
- Documents are loaded with a bounded size and no remote reference resolution.
- Batch semantics are never inferred from endpoint names; the extension (or
  external configuration) must declare them.
