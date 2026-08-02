// Package adapter implements BatchWeaver's backend adapter SDK: a versioned,
// deterministic model of backend integrations (manifests, capabilities, bindings)
// and production adapter logic — exact/composite-key PostgreSQL read-batch
// synthesis with bounded joins over database/sql contracts, Redis cluster
// hash-slot grouping, and scalar/batch contract verification.
//
// The SDK separates compile-time behavior (discovery, SQL parsing and synthesis,
// binding, diagnostics) from runtime behavior (typed batch execution and result
// mapping). Compile-time code never opens a backend connection; runtime code
// never depends on go/ast or go/types.
//
// It never infers SQL semantics from names, never concatenates raw key values
// into SQL, and rejects every query outside the explicitly supported subset with
// an exact diagnostic. Concrete pgx and go-redis bindings are public adapter
// packages wired through the same contracts.
package adapter
