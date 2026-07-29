// Package diagnostics defines the stable, dependency-free data model for
// diagnostics produced across BatchWeaver: configuration loading, the operation
// model, and future analyzers, compiler passes, and verification tooling.
//
// The types here are plain values with deterministic formatting and no
// dependency on any other BatchWeaver package, the Go tooling AST, SSA values,
// or the YAML dependency's tokens. This one-way dependency direction lets any
// layer produce and render diagnostics without creating an import cycle:
// diagnostics must not import config, operation, the root batchweaver package,
// the CLI, or compiler packages.
//
// Diagnostic codes use the reserved format described by [Code]. Once a code is
// committed and documented it must keep its meaning; see
// docs/reference/diagnostic-codes.md.
package diagnostics
