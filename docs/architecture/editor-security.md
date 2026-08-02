# Editor security

BatchWeaver's editor layer is designed to be safe in the presence of untrusted
workspaces and hostile inputs.

- **Workspace trust:** the VS Code extension performs only static text
  diagnostics in an untrusted workspace; it does not start the server, gopls, the
  daemon, tests, benchmarks, generators, or materialization until trust is
  granted.
- **No implicit mutation:** the server never writes source. Transformations are
  previewed; applying them requires an explicit, versioned editor action.
- **Child processes:** gopls is launched with an argument array (no shell),
  captured stderr, and context-bound lifetime; its stderr is redacted into logs.
- **URIs and paths:** only `file://` URIs map to files; non-file and traversing
  URIs are rejected. Virtual documents never resolve to arbitrary paths.
- **Bounded input:** JSON-RPC messages are size-bounded; malformed frames are
  rejected without panicking.
- **No telemetry:** there is no remote telemetry and no source upload. Traces are
  redacted and never include source or secrets by default.

See [the LSP threat model](../security/lsp-threat-model.md) and
[workspace trust](../security/editor-workspace-trust.md).
