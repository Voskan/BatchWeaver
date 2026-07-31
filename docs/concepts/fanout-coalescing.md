# Fan-out coalescing

Calls that already run concurrently — inside `go` statements or
`errgroup.Group.Go` closures — are lowered through the runtime bridge. The bridge
coalesces the naturally overlapping calls without introducing any new
concurrency.

```go
for _, id := range ids {
    id := id
    go func() {
        u, err := repo.GetUser(ctx, id) // -> bwopUsersGet.Call(ctx, repo, id)
        consume(u, err)
    }()
}
```

## Guarantees

- No new concurrency is introduced; the original goroutine/errgroup structure is
  unchanged.
- Each caller keeps its own context, cancellation cause, and deadline.
- Result delivery preserves the original per-goroutine semantics.
- Go 1.26 loop-variable and closure-capture semantics are preserved.

## Scope

Lowering the call inside each goroutine or `group.Go` closure is the supported
model. Aggressive parent-level static enqueue and errgroup concurrency-limit
(`SetLimit`) aware coalescing are deferred; the default preserves the existing
concurrency envelope and never bypasses group limits.
