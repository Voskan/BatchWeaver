# Configure gqlgen (GraphQL)

BatchWeaver batches resolver operations by execution wave, one BatchWeaver scope
per GraphQL operation.

## Inspect an operation

```bash
batchweaver graphql inspect --operation-file=query.graphql
batchweaver graphql graph   --operation-file=query.graphql --format=dot
```

`inspect` prints the resolver waves; `graph` renders the wave graph as text or
DOT.

## Bind resolvers

Bind a resolver field to a BatchWeaver operation through a typed declaration,
generated resolver metadata, an explicitly defined schema directive, or
configuration. Keys derive from the parent object field, a resolver argument, a
context partition, or a combination, extracted once with a typed extractor.

Register the concrete extension on the gqlgen server:

```go
import batchgqlgen "github.com/Voskan/BatchWeaver/adapters/gqlgen"

server.Use(batchgqlgen.ScopeExtension{Engine: engine})
```

Resolvers invoked by that server receive a BatchWeaver runtime scope. Use
`batchgqlgen.PartitionFromContext(ctx)` as one component of a binding partition
when selection shape must distinguish otherwise identical keys.

## DataLoader coexistence

If DataLoaders are present, choose an ownership policy (prefer-existing,
prefer-batchweaver, explicit-per-operation, or error-on-double-batching) to avoid
double queues.

## Status

The framework-neutral model, wave analysis, and concrete public-API gqlgen
extension are implemented and tested. BatchWeaver does not patch generated code
or gqlgen internals; see [limitations](../limitations/network-adapters.md).
