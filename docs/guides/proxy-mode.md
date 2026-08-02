# Proxy mode

Proxy mode is for editors that support only one Go language server. BatchWeaver
becomes the Go language server and delegates standard features to gopls.

```bash
batchweaver lsp --stdio --proxy-gopls --gopls-command=gopls
```

The editor should be configured to launch that command as its Go language server.
BatchWeaver forwards standard requests to gopls, merges capabilities and
hover/code-action/code-lens results, keeps diagnostics separated by source, and
launches gopls with an argument array (no shell). BatchWeaver never downloads or
patches gopls; a missing gopls produces a clear error.
