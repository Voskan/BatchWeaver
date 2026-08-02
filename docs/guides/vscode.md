# VS Code setup

The BatchWeaver extension source lives in `editors/vscode`. The beta VSIX is a
verified GitHub Release asset; it is not published to the Marketplace.

## Build and install

```bash
cd editors/vscode
npm ci
npm run compile
npm run package   # produces batchweaver-vscode-<version>.vsix
code --install-extension batchweaver-vscode-*.vsix
```

Ensure the `batchweaver` executable is on your PATH, or set
`batchweaver.server.path`.

## Modes

- `batchweaver.mode = "sidecar"` (default): BatchWeaver runs alongside gopls
  (started by the official Go extension).
- `batchweaver.mode = "proxy"`: BatchWeaver runs as the Go language server and
  launches gopls itself (`batchweaver.gopls.path`).

## Commands

Scan Workspace, Preview Transformation, Prove Candidate, Show Operation Graph,
Run Doctor, Restart Server, Open Logs — all under the "BatchWeaver:" prefix.

## Trust

Grant workspace trust to enable analysis; an untrusted workspace runs no server
or child process.
