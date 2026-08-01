# HTTP / OpenAPI threat model

- OpenAPI documents are loaded with a bounded size and no remote reference
  resolution by default, guarding against SSRF, path traversal, oversized
  documents, and alias-expansion bombs.
- Batch endpoints are never inferred; only explicitly declared endpoints with a
  stable correlation key are used.
- The caller's transport, TLS, cookie jar, redirect policy, and authentication
  identity are preserved; callers with different auth identities never batch.
- Authentication values, request bodies, and full URLs with identifiers are never
  logged.
- Request and response sizes and item counts are bounded and chunked.
