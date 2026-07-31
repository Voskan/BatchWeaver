# tool-exec reference

## Driver

```bash
go build -toolexec="batchweaver toolexec" ./...
```

`batchweaver toolexec <tool> [args]` runs the tool without a shell, preserves its
exit code, sets `BW_TOOLEXEC_ACTIVE` on the child to prevent recursive
`-toolexec` invocation, and strips BatchWeaver-internal environment variables.

The default architecture is overlay-first: transformations are applied through a
Go `-overlay`, so the driver delegates every tool. It never maintains a second
transformation engine.

## Diagnostics

```bash
batchweaver tool-exec doctor    # toolchain, overlay support, GOFLAGS, recursion, module
batchweaver tool-exec explain   # how the overlay and driver wire into the Go command
```

## Recursion policy

- `strip-inner` (default) — delegate safely when a marker is already present.
- `error` — fail on a detected recursive invocation.

## Exit codes

The Go command's exit code is preserved and surfaced as exit code 6
(`ExitGoCommand`), never collapsed into an internal error.
