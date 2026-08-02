// Package editor is BatchWeaver's editor-agnostic service layer. It turns a
// snapshot-bound analysis of unsaved editor buffers into deterministic
// LSP-shaped results: diagnostics, hover, code lenses, and code actions. It
// never applies edits, runs project code, or writes source; it produces
// information and preview actions that a language server surfaces to the editor.
//
// The package deliberately depends on BatchWeaver's analysis layer and the LSP
// protocol and documents packages, but not on the LSP server, transport, or
// proxy, so the compiler/runtime internals stay out of the protocol layer.
package editor
