# Editor fixture: proxy mode

This fixture demonstrates running BatchWeaver as the single Go language server in
proxy mode, delegating standard Go features to gopls.

Start the proxy from this directory:

```bash
batchweaver lsp --stdio --proxy-gopls --gopls-command=gopls
```

Configure your editor to use that command as the Go language server. gopls must
be installed and on PATH; BatchWeaver never downloads it. Standard Go features
(completion, go-to-definition, formatting) come from gopls; BatchWeaver adds
batching diagnostics, hover, lenses, and commands on top.
