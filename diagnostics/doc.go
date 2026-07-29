// Package diagnostics defines the foundational, dependency-free data model for
// diagnostics produced by BatchWeaver's future analyzers, compiler, and
// verification tooling.
//
// The types here are plain values with stable formatting and no dependency on
// compiler internals, so that any layer — from the CLI to deep analysis passes
// — can produce and render diagnostics without creating an import cycle. The
// dependency direction is strictly one-way: diagnostics must not import the CLI
// or compiler packages.
//
// Diagnostic codes use the reserved format "BWxxxx" (for example "BW0001").
// Codes are allocated deliberately as real diagnostics are introduced; this
// package documents the format without pre-allocating a large set.
package diagnostics
