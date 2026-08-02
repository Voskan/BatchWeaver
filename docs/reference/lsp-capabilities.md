# LSP capabilities reference

BatchWeaver's server advertises LSP 3.17 capabilities. It advertises only what it
implements.

| Capability | Value |
| --- | --- |
| `positionEncoding` | `utf-16` |
| `textDocumentSync` | openClose, incremental change, save |
| `hoverProvider` | yes |
| `codeActionProvider` | yes (`refactor.rewrite.batchweaver`, `source.batchweaver`) |
| `codeLensProvider` | yes |
| `executeCommandProvider` | BatchWeaver commands (see editor-commands.md) |
| `workspace.workspaceFolders` | supported (multi-root) |

Diagnostics are published via `textDocument/publishDiagnostics`. The negotiated
position encoding is UTF-16; a single canonical mapper converts UTF-16 positions
to byte offsets, correct for multibyte and emoji text.

In proxy mode, gopls capabilities are merged in: every gopls field is preserved,
and BatchWeaver's commands are unioned into `executeCommandProvider`.
