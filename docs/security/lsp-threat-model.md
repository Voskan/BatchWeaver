# LSP threat model

The editor layer processes untrusted input (editor buffers, workspace files,
JSON-RPC messages, gopls output) and can launch child processes. This model
covers the risks and mitigations.

## Threats and mitigations

- **Malicious or malformed JSON-RPC:** messages are size-bounded (64 MiB) and
  framing/parse errors are handled without panicking; the decoder is fuzzed.
- **Path traversal / non-file URIs:** only `file://` URIs map to files; non-file
  and traversing URIs are rejected. Virtual documents never resolve to arbitrary
  paths.
- **Command injection:** child processes (gopls) are launched with an argument
  array and no shell; the environment is not expanded through a shell.
- **Secret/source leakage:** no remote telemetry, no source upload; traces are
  redacted and never include source, authorization headers, operation keys, or
  request bodies by default. gopls stderr is captured into redacted logs.
- **Untrusted workspace code execution:** blocked until trust is granted (see
  workspace trust).
- **Stale or unsafe edits:** the server never writes source; applying a
  transformation requires a current proof, unchanged documents, a type-checking
  package, and explicit user action.
- **Daemon exposure:** the daemon is local-only over a `0600` Unix socket and
  exposes only health and shutdown; it executes no arbitrary methods.
- **Multiple servers:** diagnostics are separated by `source` so BatchWeaver and
  gopls never clear each other's diagnostics.

## Non-goals

No remote hosted LSP, no cloud telemetry, no browser IDE, and no modification of
the upstream gopls codebase.
