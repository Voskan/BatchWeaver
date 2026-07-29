// Package config is the reserved home for BatchWeaver's configuration schema
// and loading logic.
//
// The configuration model is deliberately not implemented in the repository
// bootstrap. Later prompts will introduce a versioned, backward-compatible
// schema that controls project discovery, analysis, transformation, and
// runtime behavior, along with loading and validation.
//
// The package is public because configuration is part of BatchWeaver's stable,
// user-facing surface. To keep that surface trustworthy, it must never expose
// internal AST, SSA, or compiler types; only plain, documented configuration
// values belong here.
package config
