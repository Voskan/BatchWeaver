# Exact-key SQL batching

An exact-key read (`SELECT ... WHERE id = $1`) issued once per item is the classic
N+1 pattern. BatchWeaver replaces N such reads with one ordered PostgreSQL query
that joins the requested keys (bound as typed arrays) against the relation and
returns one row per request ordinal.

## Guarantees

- **Order** — results map back by request ordinal.
- **Duplicates** — duplicate keys keep distinct ordinals and each receives the
  same row.
- **Missing** — a key with no row reconstructs the declared scalar missing outcome
  (`sql.ErrNoRows`).
- **Parameterization** — keys are a single array parameter; values are never
  interpolated into SQL.
- **Transaction identity** — the batch runs on the caller's `*sql.DB`, `*sql.Tx`,
  or `*sql.Conn`; the provider never acquires its own.
- **Composite identity** — multiple equality components use parallel arrays and
  unique contiguous placeholders; array arguments are passed in placeholder order.
- **Bounded join** — one qualified INNER/LEFT equality join is allowed only with
  an explicit at-most-one contract; a duplicate result ordinal fails globally.

## When it is rejected

Anything outside the supported exact/composite-key and bounded-join shapes is rejected with a precise
diagnostic rather than transformed. Use an explicit batch provider for those
cases. See [SQL synthesis](../architecture/sql-synthesis.md).
