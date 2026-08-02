// Package adapter implements BatchWeaver's backend adapter SDK: a versioned,
// deterministic model of backend integrations (manifests, capabilities, bindings)
// and the first production adapter logic — exact-key PostgreSQL read-batch
// synthesis over the standard library database/sql contracts, Redis cluster
// hash-slot grouping, and scalar/batch contract verification.
//
// The SDK separates compile-time behavior (discovery, SQL parsing and synthesis,
// binding, diagnostics) from runtime behavior (typed batch execution and result
// mapping). Compile-time code never opens a backend connection; runtime code
// never depends on go/ast or go/types.
//
// It never infers SQL semantics from names, never concatenates raw key values
// into SQL, and rejects every query outside the explicitly supported subset with
// an exact diagnostic. Concrete client bindings for pgx and go-redis are declared
// in the manifest and are wired through the same contracts; see
// docs/limitations/backend-adapters.md for their status.
package adapter
