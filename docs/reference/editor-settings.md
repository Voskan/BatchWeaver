# Editor settings reference

VS Code settings (also documented in the extension's `package.json`):

| Setting | Default | Meaning |
| --- | --- | --- |
| `batchweaver.enabled` | `true` | Enable the language server. |
| `batchweaver.mode` | `sidecar` | `sidecar` or `proxy`. |
| `batchweaver.server.path` | `batchweaver` | Path to the BatchWeaver executable. |
| `batchweaver.gopls.path` | `gopls` | gopls path (proxy mode). |
| `batchweaver.deepAnalysis` | `false` | Allow deep proof/verification analysis (trusted workspace only). |
| `batchweaver.trace.server` | `off` | LSP trace verbosity (redacted). |

## Precedence

```text
hard safety            (cannot be overridden)
workspace BatchWeaver config
editor workspace settings
editor user presentation settings
command-specific explicit flags
```

Security-sensitive behavior requires workspace configuration or explicit consent,
never a user-global presentation setting alone.
