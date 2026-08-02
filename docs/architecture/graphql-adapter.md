# GraphQL adapter

The GraphQL adapter batches resolver operations by execution wave. It keeps a
framework-neutral operation model and resolver-wave analysis
([ADR 0053](../adr/0053-gqlgen-extension-integration.md)). The concrete
`adapters/gqlgen` extension uses gqlgen's public operation and field interceptor
APIs to establish a runtime scope and attach normalized field partitions.

## Model

`internal/adapter` provides a framework-neutral GraphQL model — document,
operation, field (alias, directives, selection set), fragment definitions,
fragment spreads, and inline fragments — built by a recursive-descent parser
(no regex). From it, `ResolverWaves` computes execution waves and
`NormalizeSelectionDigest` produces an alias-independent selection digest.

## Scope and waves

One BatchWeaver scope is established per GraphQL operation
([ADR 0052](../adr/0052-graphql-operation-scope.md)). Wave 0 is the operation's
top-level fields; each subsequent wave holds the fields of the previous wave's
selection sets. Fields ready at the same dependency frontier are batched together
when they bind to the same operation and share a compatible partition.

## Partitioning and preservation

Resolver calls partition by normalized selection digest and authorization/tenant
context ([ADR 0054](../adr/0054-graphql-selection-partitioning.md)). Field paths
(alias-aware), error paths/extensions, per-field vs operation errors, null
propagation, and list ordering are preserved
([ADR 0055](../adr/0055-graphql-error-nullability.md)). Directives that change
behavior are barriers; mutations are not reordered; subscriptions use bounded
per-event scopes.

## CLI

```bash
batchweaver graphql inspect --operation-file=query.graphql
batchweaver graphql graph   --operation-file=query.graphql --format=dot
```
