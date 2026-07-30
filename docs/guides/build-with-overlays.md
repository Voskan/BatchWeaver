# Build, test, and run with overlays

`batchweaver build`, `test`, and `run` execute transformed code through a Go
`-overlay` without editing your source tree.

## Build

```bash
batchweaver build ./cmd/service
```

BatchWeaver plans the module, validates the transformed packages, writes an
overlay, and runs `go build -overlay=<overlay.json> ./cmd/service`. The Go
command's exit code is preserved and the working tree is unchanged.

## Test

```bash
batchweaver test ./...
```

Delegates to `go test` against the same overlay. Standard flags (`-run`,
`-count`, `-race`, `-cover`, `-coverprofile`, `-tags`, `-timeout`, `-short`,
`-fuzz`, `-fuzztime`, `-bench`, `-benchtime`, `-cpu`, `-v`, `-json`) are forwarded
to `go test`; BatchWeaver does not reimplement their semantics.

## Run

```bash
batchweaver run ./cmd/example -- --example-flag=value
```

Application arguments after `--` are forwarded to the program.

## Notes

- If no candidate is transformable, the commands run the Go tool with no overlay.
- Transformed bytes live under the ignored `.batchweaver/` cache; source is never
  modified by these commands.
