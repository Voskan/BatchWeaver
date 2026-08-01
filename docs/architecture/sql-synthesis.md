# SQL synthesis

BatchWeaver synthesizes a batch query only for a deliberately narrow, formally
validated exact-key SELECT shape ([ADR 0044](../adr/0044-narrow-exact-key-sql-synthesis.md)).

## Parser

A hand-written tokenizer (skipping comments, handling quoted identifiers,
placeholders, and strings) feeds a strict recursive parser. It never uses regex
as the primary parser and never panics on input. Any construct outside the
supported grammar is rejected with an exact code, node, and byte offset.

## Supported shape

```sql
SELECT <explicit columns>
FROM <single relation> [alias]
WHERE <key column> = $N
  [AND <column> IS [NOT] NULL]   -- key-independent only
```

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

Keys are bound as one typed array parameter — never interpolated. The result is
mapped by request ordinal, preserving order, duplicates, and the declared missing
outcome (`sql.ErrNoRows`). Larger inputs are chunked deterministically.

## Rejected by default

`SELECT *`, function calls (potential volatility), joins, `GROUP BY`, `HAVING`,
`ORDER BY`, `LIMIT`/`OFFSET`, window/set operations, CTEs, `OR`/`IN`, locking
clauses, writes (INSERT/UPDATE/DELETE/MERGE), multiple statements, and non-parameter
key comparisons. See [the SQL support matrix](../reference/sql-support-matrix.md).
