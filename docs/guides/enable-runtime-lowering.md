# Enable runtime lowering

Runtime lowering rewrites certified scalar calls to route through the BatchWeaver
runtime. The transformed module must depend on the `github.com/Voskan/BatchWeaver`
module so the generated bridge can import the `bridge` package.

## Plan

```bash
batchweaver transform plan --strategy=runtime-call-coalescing ./...
```

Add `fanout-coalescing` and `static-sibling-fusion` to the comma-separated list
to lower concurrent and straight-line sibling call sites. The plan reports the
planned lowerings, generated bridge files, and validation, and changes no source.

## Build, test, run

```bash
batchweaver build --strategy=runtime-call-coalescing -- ./cmd/service
batchweaver test  --strategy=runtime-call-coalescing -- -race ./...
batchweaver run   --strategy=runtime-call-coalescing -- ./cmd/example -- --flag=v
```

BatchWeaver flags precede `--`; Go and application arguments follow it. Execution
uses a Go `-overlay`; the working tree is never modified.

## Activate coalescing at runtime

A lowered call falls back to the scalar call unless the application installs a
typed bound operation into the scope context:

```go
ctx = bridge.WithOperation(ctx, "users.get", boundUsersGet)
```

Declare the batch provider and bind the operation with the typed runtime, then
install it for the scope. Without this, lowered calls behave exactly like the
original scalar calls.

## Inspect

```bash
batchweaver runtime inspect --operation=users.get ./...
batchweaver barrier inspect ./internal/orders
batchweaver tool-exec doctor
```
