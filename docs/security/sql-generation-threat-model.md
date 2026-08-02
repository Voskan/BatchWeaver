# SQL generation threat model

Generated SQL and adapter behavior treat scalar SQL and keys as untrusted.

- **Parameterized only** — synthesized queries bind one typed array per key
  component. Raw key values are never concatenated into SQL.
- **Validated identifiers** — table and column identifiers come only from parsing
  a static, compile-time scalar query; runtime table/column names are never
  accepted.
- **Bounded output** — generated statements and batches are bounded and chunked to
  respect backend and driver parameter limits.
- **No untrusted fragments** — SQL assembled from runtime fragments is rejected
  (`BW6101`); the parser rejects multiple statements and unsupported syntax.
- **Redaction** — diagnostics and observability redact values and raw keys; query
  text logging is opt-in.
- **Redis** — raw keys are never logged by default; observability uses bounded
  labels and keyed fingerprints, and credentials remain client-owned.
- **No connections at compile time** — parsing and synthesis never open a backend
  connection.
- **Cardinality defense** — joins require an explicit at-most-one contract and
  the provider rejects duplicate ordinals instead of silently choosing a row.
- **Plan integrity** — query, key types/placeholders, projection, cardinality,
  and result contract are covered by a SHA-256 plan digest checked before I/O.
