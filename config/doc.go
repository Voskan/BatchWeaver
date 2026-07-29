// Package config implements BatchWeaver's strict, versioned configuration
// system: schema version 1, YAML and JSON loading, deterministic includes and
// merge, centralized defaults, normalization into an operation catalog, semantic
// validation, canonical rendering, and a semantic digest.
//
// Loading is deterministic and side-effect free: it never reads home-directory
// or environment configuration implicitly, forbids remote includes, bounds input
// sizes, and rejects unknown fields and duplicate keys. Invalid user
// configuration is reported as diagnostics rather than opaque errors; see Load.
//
// Dependency direction: config may import operation and diagnostics (and the
// internal configuration-implementation packages). Its public API never exposes
// internal decoder or node types.
package config
