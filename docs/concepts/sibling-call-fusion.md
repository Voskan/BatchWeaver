# Sibling-call fusion

Straight-line sibling calls to the same operation are lowered through the runtime
bridge in lexical order:

```go
author, err := users.GetUser(ctx, post.AuthorID)
if err != nil { return err }
editor, err := users.GetUser(ctx, post.EditorID)
if err != nil { return err }
```

Each call becomes `bwopUsersGet.Call(ctx, users, post.AuthorID)` and so on. The
rewrite preserves source evaluation order, first-error behavior, receiver and
context stability, variable scopes, and defer/panic order. When the calls run in
one scope the runtime coalesces them; otherwise each falls back to the scalar
call.

This stage lowers siblings through the runtime bridge (the "immediate runtime
call" model). A fully static sibling batch with explicit wave coordination is a
future refinement; the runtime-bridge model is always safe and never reorders
observable effects.
