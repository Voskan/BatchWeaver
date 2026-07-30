# Build overlays

BatchWeaver executes transformed code without editing the source tree by using
the standard Go command `-overlay` mechanism.

## How it works

1. A plan is built and saved under `.batchweaver/cache/transform/<plan-id>/`.
   Transformed files are stored content-addressed under `files/`.
2. An overlay manifest (`overlay.json`) maps each original absolute source path to
   its transformed backing file.
3. `batchweaver build|test|run` invokes the installed Go tool with
   `-overlay=<overlay.json>` and the user's arguments, preserving the Go command's
   exit code and streaming its output.

## Consistency

The same transformed bytes back both in-process package loading
(`packages.Config.Overlay`, used for type-check validation) and the Go command
(`-overlay`), so analysis, type checking, and compilation never diverge. The
overlay manifest digest is derived from workspace-relative paths and transformed
digests.

## Privacy and safety

- Transformed files and overlays live under the ignored `.batchweaver/` state
  directory and are never uploaded or transmitted.
- The Go command is run with an explicit argument array and no shell; BatchWeaver
  environment variables are stripped from the child to prevent recursive wrapper
  invocation.
- The working tree is never modified by build, test, or run.

See [ADR 0030](../adr/0030-overlays-default-non-mutating.md).
