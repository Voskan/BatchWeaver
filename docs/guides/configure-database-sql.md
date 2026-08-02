# Configure the database/sql adapter

The `database/sql` adapter binds explicit scalar/batch operations and synthesizes
exact/composite-key PostgreSQL read batches, with one optional bounded join.

## Inspect a synthesized query

```bash
batchweaver adapter inspect \
  --sql="SELECT id, name FROM users WHERE id = \$1" \
  --key-type=bigint
```

This prints the generated ordered batch query and the result contract, or a
precise rejection for an unsupported query.

## Composite key and bounded join

```bash
batchweaver adapter inspect \
  --sql='SELECT u.tenant_id, u.id, p.display_name
         FROM users u
         LEFT JOIN profiles p ON p.user_id = u.id
         WHERE u.tenant_id = $1 AND u.id = $2' \
  --key-types=uuid,bigint \
  --join-cardinality=at-most-one
```

The placeholders must be unique and contiguous from `$1`. Every joined column
must be qualified. `at-most-one` is an explicit application/schema contract;
BatchWeaver does not connect to PostgreSQL to infer it.

## Create a non-mutating compiler plan

```bash
batchweaver adapter plan-sql \
  --sql='SELECT tenant_id, id, name FROM users
         WHERE tenant_id = $1 AND id = $2' \
  --key-types=uuid,bigint \
  --package-name=queries \
  --package-path=example.com/service/queries \
  --output=queries/batchweaver_users_sql_gen.go \
  --constant=GetUsersBatchSQL \
  --operation=users.get
```

This writes only a content-addressed plan and Go overlay beneath ignored
`.batchweaver/` state. The generated binding is parsed and type-checked. Inspect,
diff, verify, or explicitly materialize it with the normal `transform` commands.

## Explain a rejection

```bash
batchweaver adapter explain --sql="SELECT id, now() FROM users WHERE id = \$1"
```

## Verify a binding

```bash
batchweaver adapter verify
```

## Runtime binding

`SQLProvider.KeyArgs` must return one array per composite component in
placeholder order. The provider keeps the caller's `*sql.DB`, `*sql.Tx`, or
`*sql.Conn`, chunks deterministically, checks the plan digest before I/O, and
rejects duplicate/out-of-range ordinals. Default limits remain conservative.
