// Package proxy implements BatchWeaver's optional gopls-compatible LSP proxy.
// The editor connects to BatchWeaver, which launches (or connects to) the user's
// installed gopls process, forwards standard Go language traffic to it, and
// merges BatchWeaver's own capabilities, diagnostics, hover, code lenses, and
// code actions on top.
//
// The proxy never imports gopls internal packages and never patches the user's
// gopls installation; it speaks only the public LSP wire protocol. Request-ID
// namespacing between the two peers is handled automatically because forwarded
// requests are re-issued through the destination connection, which assigns and
// matches its own IDs. See docs/architecture/gopls-proxy.md and
// docs/adr/0073-optional-lsp-proxy.md.
package proxy
