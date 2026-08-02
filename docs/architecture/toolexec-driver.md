# The -toolexec driver

BatchWeaver integrates with the Go build in an overlay-first architecture: the
CLI applies transformations through a Go `-overlay`, and an optional `-toolexec`
driver observes compile actions and delegates every tool faithfully.

## Why overlay-first

The overlay is the single source of transformation truth. The
`-toolexec` driver does not maintain a second transformation engine; it provides
an integration hook, recursion safety, and environment hygiene. See
[ADR 0037](../adr/0037-toolexec-overlay-first.md).

## Driver behavior

`batchweaver toolexec <tool> [args]` is invoked by the Go command as
`go build -toolexec="batchweaver toolexec" ...`. The driver:

- runs the tool with `exec.CommandContext` and an explicit argument array — never
  a shell;
- preserves stdout, stderr, and the exit code;
- sets a private marker (`BW_TOOLEXEC_ACTIVE`) on the child to detect recursion;
- strips BatchWeaver-internal environment variables from the child.

`IsCompile` identifies compiler actions for callers that transform only compile
steps.

## Diagnostics

- `batchweaver tool-exec doctor` checks the toolchain, overlay support, GOFLAGS,
  recursion markers, and the workspace module, and prints actionable status.
- `batchweaver tool-exec explain` describes exactly how the overlay and driver
  wire into the Go command.

## Recursion policy

The default policy delegates safely when a marker is already present (a nested Go
command). An error policy is available for strict environments. See
[the tool-exec reference](../reference/tool-exec.md).
