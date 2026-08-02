# SQL support matrix (synthesis)

| Construct | Status |
| --- | --- |
| `SELECT` with explicit columns | supported |
| base relation with optional alias | supported |
| `WHERE key = $1 [AND key2 = $2 ...]` | supported; unique contiguous placeholders |
| key-independent `AND col IS [NOT] NULL` | supported |
| one qualified `INNER JOIN` / `LEFT JOIN` equality | supported with explicit `at-most-one` contract |
| `SELECT *` | rejected (BW6105) |
| function calls (e.g. `now()`) | rejected (BW6103) |
| join without at-most-one evidence | rejected (BW6106) |
| multiple/right/full/cross/lateral/USING joins | rejected (BW6102/BW6107) |
| `GROUP BY` / `HAVING` / window | rejected (BW6102) |
| `ORDER BY` / `LIMIT` / `OFFSET` | rejected (BW6102) |
| set operations / CTEs | rejected (BW6102) |
| `OR` / `IN` predicates | rejected (BW6102) |
| locking (`FOR UPDATE`, ...) | rejected (BW6102) |
| INSERT / UPDATE / DELETE / MERGE | rejected (BW6102) |
| non-parameter key comparison | rejected (BW6104) |
| multiple statements | rejected (BW6102) |

Dialect: PostgreSQL. Rejections carry an exact code, node, and byte offset.
Generated synthesis plans also carry an integrity digest and are revalidated by
the runtime provider before any query is executed.
