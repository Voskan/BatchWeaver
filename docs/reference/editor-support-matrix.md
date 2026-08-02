# Editor support matrix

"Standard LSP" means the feature works through the protocol for any conformant
client. Community entries are protocol-validated, not officially maintained
integrations. A machine-readable copy is in `editor-support-matrix.json`.

| Editor | Sidecar | Proxy | Diagnostics | Code actions | CodeLens | Hover | Commands | Extension features |
| --- | --- | --- | --- | --- | --- | --- | --- | --- |
| VS Code | yes | yes | yes | yes | yes | yes | yes | status bar, output, virtual docs |
| Neovim (built-in LSP) | yes | yes | yes | yes | yes | yes | yes | via standard LSP |
| Emacs / Eglot | via multi-server | yes | yes | yes | yes | yes | yes | via standard LSP |
| Helix | via config | yes | yes | yes | yes | yes | yes | via standard LSP |
| Zed | community | community | yes | yes | yes | yes | yes | via standard LSP |

All rows depend on client support for the specific LSP feature; BatchWeaver only
advertises what it implements.

The official extension runs real headless activation/command-registration tests
on the minimum VS Code 1.85.2 host and the current pinned 1.131.0 host. gopls
proxy compatibility uses a real v0.21.1 process. Neovim, Eglot, Helix, and Zed
remain community protocol configurations and are not represented as maintained
host-UI test results.
