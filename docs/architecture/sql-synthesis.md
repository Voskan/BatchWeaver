# SQL synthesis

BatchWeaver synthesizes a batch query only for a deliberately narrow, formally
validated exact/composite-key SELECT shape, optionally carrying one bounded join
([ADR 0044](../adr/0044-narrow-exact-key-sql-synthesis.md)).

## Parser

A hand-written tokenizer (skipping comments, handling quoted identifiers,
placeholders, and strings) feeds a strict recursive parser. It never uses regex
as the primary parser and never panics on input. Any construct outside the
supported grammar is rejected with an exact code, node, and byte offset.

## Supported shape

```sql
SELECT <explicit columns>
FROM <single relation> [alias]
WHERE <key column> = $1
  [AND <key column> = $2 ...]   -- contiguous, unique placeholders
  [AND <column> IS [NOT] NULL]   -- key-independent only
```

The base relation may have one qualified `INNER JOIN` or `LEFT JOIN` equality.
Every projected/filter column must then be qualified, the key must belong to the
base relation, and synthesis requires an explicit `at-most-one` cardinality
contract. BatchWeaver does not infer uniqueness from SQL text or a live schema.

## Generated PostgreSQL batch (exact-key)

```sql
WITH bw_requested(bw_key, bw_ord) AS (
    SELECT * FROM unnest($1::bigint[]) WITH ORDINALITY
)
SELECT bw_requested.bw_ord, users.id, users.name
FROM bw_requested
LEFT JOIN users ON users.id = bw_requested.bw_key
ORDER BY bw_requested.bw_ord
```

Each key component is bound as a typed array parameter — never interpolated. The result is
mapped by request ordinal, preserving order, duplicates, and the declared missing
outcome (`sql.ErrNoRows`). Larger inputs are chunked deterministically.

Composite keys use parallel arrays with one deterministic placeholder per key
component. Generated plans are content-addressed; the runtime rejects a modified
plan, an out-of-range ordinal, or a duplicate ordinal (which would violate the
declared scalar/join cardinality).

`adapter plan-sql` generates a Go constant and plan into the ignored
`.batchweaver` cache, type-checks it through a Go overlay, and produces a source
map. It never writes the requested source file unless the separate explicit
materialization workflow is used.

## Rejected by default

`SELECT *`, function calls (potential volatility), one-to-many/multiple/right/
full/cross/lateral joins, `GROUP BY`, `HAVING`,
`ORDER BY`, `LIMIT`/`OFFSET`, window/set operations, CTEs, `OR`/`IN`, locking
clauses, writes (INSERT/UPDATE/DELETE/MERGE), multiple statements, and non-parameter
key comparisons. See [the SQL support matrix](../reference/sql-support-matrix.md).
