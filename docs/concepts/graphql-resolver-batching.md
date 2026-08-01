# GraphQL resolver batching

A GraphQL query that resolves a list and then a field per element is a resolver
N+1 pattern. BatchWeaver establishes one scope per operation and batches the
per-element resolver calls within an execution wave.

## Waves

Resolvers ready after the same dependency frontier form a wave. Wave 0 is the
operation's top-level fields; wave 1 resolves the fields of their results; and so
on. Fields in a wave that bind to the same operation and share a compatible
partition are coalesced into one batch call.

## What is preserved

Field paths (alias-aware), fragments and inline fragments, directive behavior,
per-field and operation errors, null propagation, and list ordering. Fields are
never returned to a caller that did not request them.

## What is not batched

Different operations, requests, users, tenants, or authorization scopes; top-level
mutations; and fields whose directives or selection-dependent providers are
incompatible.
