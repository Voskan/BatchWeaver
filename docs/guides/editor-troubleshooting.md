# Editor troubleshooting

Run the doctor first:

```bash
batchweaver editor doctor
```

It reports the BatchWeaver, Go, and gopls versions, the LSP protocol version, the
platform, and daemon status.

Common issues:

- **No BatchWeaver diagnostics:** confirm the workspace is a Go module, the
  `batchweaver` executable is on PATH, and (in VS Code) the workspace is trusted.
- **Proxy mode fails to start:** gopls must be installed and on PATH, or set
  `--gopls-command` / `batchweaver.gopls.path`. BatchWeaver never downloads gopls.
- **No output on stdout:** in stdio mode the server writes only protocol messages
  to stdout; logs go to stderr. Redirect stderr to a file to inspect them.
- **Stale results after edits:** diagnostics are debounced; a brief delay after
  typing is expected. Results from a superseded snapshot are dropped, never shown.
