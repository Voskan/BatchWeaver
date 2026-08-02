# Composite-key and bounded-join SQL

This compile-tested example parses a static PostgreSQL query with a composite
`(tenant_id, id)` key and one left join, requires explicit `at-most-one`
cardinality, and validates the content-addressed batch plan.

Inspect the generated SQL:

```bash
go test ./examples/composite-join-sql
batchweaver adapter inspect \
  --sql='SELECT u.tenant_id, u.id, p.display_name FROM users u LEFT JOIN profiles p ON p.user_id = u.id WHERE u.tenant_id = $1 AND u.id = $2' \
  --key-types=uuid,bigint \
  --join-cardinality=at-most-one
```

The runtime receives the two key arrays in placeholder order. Duplicate rows
for one request ordinal are rejected because they contradict the declared
scalar cardinality.
