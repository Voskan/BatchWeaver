# Helix setup (community configuration)

Helix uses one language server set per language. Add BatchWeaver in proxy mode via
`languages.toml` (community configuration; protocol-validated, not an official
integration):

```toml
[[language]]
name = "go"
language-servers = ["batchweaver"]

[language-server.batchweaver]
command = "batchweaver"
args = ["lsp", "--stdio", "--proxy-gopls", "--gopls-command=gopls"]
```
