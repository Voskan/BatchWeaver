// Package operation defines BatchWeaver's canonical operation domain model: the
// stable identifiers, Go symbol references, semantic policies, and contracts
// that describe how a scalar operation relates to its batch equivalent.
//
// The model is data and type safety only. Nothing in this package intercepts
// calls, schedules work, or performs batching; those behaviors belong to later
// runtime and compiler prompts. Values here are immutable by convention:
// construct them through the provided constructors and pass them by value rather
// than mutating fields.
//
// Dependency direction: operation may import diagnostics and the standard
// library only. It must not import config, the root batchweaver package, the
// CLI, or compiler packages.
package operation
