# Zed setup (community configuration)

Zed's Go support is provided by gopls. Running BatchWeaver as an additional or
proxy language server in Zed depends on your Zed version's LSP configuration
support and may require an extension. BatchWeaver speaks standard LSP over stdio:

```bash
batchweaver lsp --stdio --proxy-gopls --gopls-command=gopls
```

This is a community configuration and is not an official Zed integration.
