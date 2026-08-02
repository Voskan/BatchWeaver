# Concrete client bindings

This compile-tested example verifies the public integration surface for pgx v5,
go-redis v9, gqlgen, and grpc-go without requiring external services.

```bash
go test ./examples/client-bindings -count=1
go test ./adapters/... -count=1
```

The adapter suite adds pgxmock row-contract coverage, real go-redis calls against
miniredis, gqlgen public extension lifecycle tests, and a real grpc-go bufconn
RPC. Production deployments should additionally run the same application tests
against their actual PostgreSQL, Redis/Redis Cluster, gqlgen schema, and gRPC
service versions.
