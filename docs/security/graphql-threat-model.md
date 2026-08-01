# GraphQL threat model

- Cross-user, cross-tenant, and cross-authorization batching is prevented by
  partitioning resolver calls by authorization/tenant context and normalized
  selection.
- A field is never returned to a caller that did not request it or is not
  authorized for it, even when another caller in the same wave requested it.
- Query variables, documents, and field values are never logged; observability
  uses normalized schema field identity, not high-cardinality response paths.
- Framework complexity, depth, and resolver limits are respected; batching never
  bypasses them.
- Subscriptions use bounded per-event scopes; operation-scoped memoization is not
  retained indefinitely.
