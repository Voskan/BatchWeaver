// Package adaptive implements BatchWeaver's production optimization and control
// layer: privacy-safe workload profiling, versioned cost models, a bounded and
// explainable adaptive scheduler controller, multi-operation execution waves,
// recursive breadth-first batching for proven traversals, fairness and tenant
// quotas, overload detection with admission control and load shedding, and
// deterministic offline replay, simulation, and reporting.
//
// The package is deliberately conservative. Every online behavior is gated on a
// valid operation contract, a valid semantic proof, valid runtime/adapter ABIs,
// current and compatible profiles, hard configuration bounds, and SLO
// guardrails (see the Core Adaptive Safety Rule in the design). Adaptive logic
// can only recommend settings, or apply settings that stay within
// caller-configured hard bounds; it can never combine partitions, change
// transaction identity, retry unsafe writes, exceed backend limits, bypass
// barriers, or override an operator's emergency disablement.
//
// Determinism: all offline artifacts (profiles, decisions, wave graphs, reports,
// replay outputs) are content-addressed and serialize deterministically.
// Time-dependent logic reads a Clock so tests use a fake clock and never sleep.
//
// Privacy: profiles never store raw operation keys, tokens, tenant names,
// request bodies, SQL parameters, GraphQL variables, gRPC metadata values, HTTP
// headers, raw URLs, or source code. Cardinality is bounded; tenants appear only
// as anonymized, keyed-hash classes.
//
// This package adds no third-party dependencies: histograms, sketches, cost
// models, and controllers are implemented over the standard library.
package adaptive
