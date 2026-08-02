# BatchWeaver for VS Code

Live batching diagnostics, proof-aware code actions, transformation previews, and
operation views for Go, powered by the BatchWeaver language server.

BatchWeaver is **not** a gopls plugin and does not modify your gopls
installation. It runs in one of two modes:

- **Sidecar** (default): BatchWeaver runs alongside gopls. Use when your editor
  supports multiple Go language servers.
- **Proxy**: BatchWeaver runs as the Go language server and delegates standard
  Go features to gopls over the public LSP protocol.

## Requirements

- The `batchweaver` executable on your PATH (or set `batchweaver.server.path`).
- `gopls` for standard Go features (proxy mode launches it; sidecar mode expects
  your existing Go extension to run it). BatchWeaver never downloads gopls.

## Building from source

This extension is distributed as source. Build it with a pinned toolchain:

```bash
cd editors/vscode
npm ci          # generates node_modules from package.json (network required)
npm run lint
npm run typecheck
npm run compile
npm run package # produces batchweaver-vscode.vsix
```

The committed `package-lock.json` pins the extension dependency graph. The beta
VSIX is distributed as a verified GitHub Release asset; it is not published to
the Visual Studio Marketplace.

## Workspace trust

In an untrusted workspace the extension does not start the server, gopls, or any
child process. Grant trust to enable BatchWeaver.

## Privacy

No remote telemetry. No source upload. The extension writes no source files;
transformations are previewed and applied only through explicit, versioned
editor actions.
