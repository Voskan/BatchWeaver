# Neovim setup (community configuration)

BatchWeaver speaks standard LSP over stdio, so any LSP client can use it. The
snippet below is a community configuration; it is protocol-validated but not part
of an official Neovim integration.

## Sidecar (with gopls configured separately)

```lua
vim.api.nvim_create_autocmd("FileType", {
  pattern = "go",
  callback = function()
    vim.lsp.start({
      name = "batchweaver",
      cmd = { "batchweaver", "lsp", "--stdio" },
      root_dir = vim.fs.dirname(vim.fs.find({ "go.mod" }, { upward = true })[1]),
    })
  end,
})
```

## Proxy (BatchWeaver as the single Go server)

```lua
vim.lsp.start({
  name = "batchweaver-proxy",
  cmd = { "batchweaver", "lsp", "--stdio", "--proxy-gopls", "--gopls-command=gopls" },
  root_dir = vim.fs.dirname(vim.fs.find({ "go.mod" }, { upward = true })[1]),
})
```
