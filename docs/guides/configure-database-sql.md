# Configure the database/sql adapter

The `database/sql` adapter binds explicit scalar/batch operations and synthesizes
exact-key PostgreSQL read batches.

## Inspect a synthesized query

```bash
batchweaver adapter inspect \
  --sql="SELECT id, name FROM users WHERE id = \$1" \
  --key-type=bigint
```

This prints the generated ordered batch query and the result contract, or a
precise rejection for an unsupported query.

## Explain a rejection

```bash
batchweaver adapter explain --sql="SELECT id, now() FROM users WHERE id = \$1"
```

## Verify a binding

```bash
batchweaver adapter verify
```

## Configuration

```yaml
adapters:
  database_sql:
    enabled: true
    dialect: postgres
    synthesis:
      exact_key_reads: true
      composite_keys: false
      joins: false
    limits:
      max_items: 1000
      max_parameters: 32000
      max_payload_bytes: 4194304
      max_concurrency: 4
```

Defaults are conservative and validated strictly.
