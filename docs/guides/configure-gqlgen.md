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

## DataLoader coexistence

If DataLoaders are present, choose an ownership policy (prefer-existing,
prefer-batchweaver, explicit-per-operation, or error-on-double-batching) to avoid
double queues.

## Status

The framework-neutral model and wave analysis are implemented and tested. The
concrete gqlgen runtime hook is deferred in this build; see
[limitations](../limitations/network-adapters.md).
