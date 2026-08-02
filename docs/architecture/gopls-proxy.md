# gopls proxy

The optional proxy (`internal/lsp/proxy`) lets a client that supports only one Go
language server use BatchWeaver and gopls together. The editor connects to
BatchWeaver, which launches the user's installed gopls and composes the two.

BatchWeaver never imports gopls internal packages and never patches gopls; it
speaks only the public LSP wire protocol (see
[ADR 0073](../adr/0073-optional-lsp-proxy.md)).

## Request-ID namespacing

Client and gopls request IDs can collide. The proxy avoids a manual mapping table
by re-issuing each forwarded request through the destination connection, which
assigns and matches its own IDs. Responses, cancellation, and server-initiated
requests (configuration, applyEdit, progress) are routed by the same mechanism in
both directions.

## Routing and merging

- **Forwarded to gopls:** standard Go requests and notifications.
- **Handled by BatchWeaver:** `batchweaver.*` commands.
- **Merged:** `initialize` (BatchWeaver capabilities are overlaid onto gopls's,
  preserving every gopls field and unioning execute-command lists), `hover`
  (gopls hover plus a BatchWeaver section), `codeAction`, and `codeLens`
  (concatenated).
- **Diagnostics** keep their own `source` (`batchweaver` vs gopls), so neither
  server clears the other's diagnostics.

## gopls lifecycle

The proxy discovers gopls by configured command/path, captures its stderr into
redacted logs, and shuts it down with the session. It never downloads gopls; a
missing gopls yields a clear error.
