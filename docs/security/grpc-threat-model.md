# gRPC threat model

- Authorization tokens, credentials, and routing metadata are never merged across
  callers; such keys partition so only identical-context callers share a batch.
- Per-RPC credentials, transport security, and channel identity partition calls.
- Messages are bounded and chunked to respect send/receive size limits.
- Interceptors are not bypassed; unknown interceptor semantics reject aggressive
  fusion.
- Metadata and headers are never logged; binary metadata requires explicit
  handling.
