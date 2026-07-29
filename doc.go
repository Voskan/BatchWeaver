// Package batchweaver is the user-facing root of BatchWeaver's foundational
// contracts. It provides the canonical generic batch request and response
// types, the typed scalar and batch function contracts, and the typed
// declarations that connect an operation spec to concrete implementations.
//
// Everything here is data and type safety. Declaring an operation does not yet
// cause any call to be intercepted, batched, scheduled, or deduplicated; those
// behaviors arrive in later runtime and compiler prompts. Declarations are
// plain package-level values with no global registration, which keeps them
// statically discoverable by a future analyzer.
//
// Dependency direction: this package may import operation and the standard
// library. It must not import config, the CLI, or compiler packages.
package batchweaver
