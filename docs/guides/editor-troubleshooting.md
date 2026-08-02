# Editor troubleshooting

Run the doctor first:

```bash
batchweaver editor doctor
```

It reports the BatchWeaver, Go, and gopls versions, the LSP protocol version, the
platform, and daemon status.

The daemon is optional. Start it in the workspace to share analysis between CLI
and LSP processes, inspect privacy-safe counters, or remove its state:

```bash
batchweaver daemon start
batchweaver daemon status
batchweaver daemon stop
batchweaver daemon clean
```

If the daemon is absent, stale, or incompatible, editor requests fall back to
in-process analysis. LSP stderr logs show only cache source and hit status; no
source or cache key is logged.

Common issues:

- **No BatchWeaver diagnostics:** confirm the workspace is a Go module, the
  `batchweaver` executable is on PATH, and (in VS Code) the workspace is trusted.
- **Proxy mode fails to start:** gopls must be installed and on PATH, or set
  `--gopls-command` / `batchweaver.gopls.path`. BatchWeaver never downloads gopls.
- **No output on stdout:** in stdio mode the server writes only protocol messages
  to stdout; logs go to stderr. Redirect stderr to a file to inspect them.
- **Stale results after edits:** diagnostics are debounced; a brief delay after
  typing is expected. Results from a superseded snapshot are dropped, never shown.
