# SQL support matrix (synthesis)

| Construct | Status |
| --- | --- |
| `SELECT` with explicit columns | supported |
| single relation with optional alias | supported |
| `WHERE key = $N` (single equality) | supported |
| key-independent `AND col IS [NOT] NULL` | supported |
| `SELECT *` | rejected (BW6105) |
| function calls (e.g. `now()`) | rejected (BW6103) |
| joins | rejected (BW6102) |
| `GROUP BY` / `HAVING` / window | rejected (BW6102) |
| `ORDER BY` / `LIMIT` / `OFFSET` | rejected (BW6102) |
| set operations / CTEs | rejected (BW6102) |
| `OR` / `IN` predicates | rejected (BW6102) |
| locking (`FOR UPDATE`, ...) | rejected (BW6102) |
| INSERT / UPDATE / DELETE / MERGE | rejected (BW6102) |
| non-parameter key comparison | rejected (BW6104) |
| multiple statements | rejected (BW6102) |

Dialect: PostgreSQL. Rejections carry an exact code, node, and byte offset.
