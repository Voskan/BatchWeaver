package adapter

import "sort"

// RuntimeABIVersion is the bridge ABI the adapters target. It must match the
// bridge package; a mismatch invalidates adapter bindings.
const RuntimeABIVersion = "batchweaver.bridge/v1alpha1"

// Adapter categories.
const (
	CategoryBackend = "backend"
	CategoryNetwork = "network"
)

// builtinManifests is the immutable set of adapter manifests compiled into this
// build. There is no mutable global registry; callers receive copies.
func builtinManifests() []Manifest {
	ms := []Manifest{
		{
			SchemaVersion: SchemaVersion, AdapterID: "database/sql", Version: 1,
			DisplayName: "database/sql", Category: CategoryBackend, Status: StatusReady, RuntimeABI: RuntimeABIVersion,
			Dialects: []string{"postgres"},
			Capabilities: []Capability{
				CapExplicitBatchBinding, CapExactKeyReadSynthesis, CapCompositeKeyRead, CapBoundedJoinRead, CapOrderedResultMapping,
				CapSparseResultMapping, CapTransactionPartition, CapChunking,
				CapSemanticVerification,
			},
			Notes: "Exact and composite-key PostgreSQL read synthesis with bounded at-most-one joins over the standard library.",
		},
		{
			SchemaVersion: SchemaVersion, AdapterID: "pgx", Version: 1,
			DisplayName: "pgx", Category: CategoryBackend, Status: StatusReady, RuntimeABI: RuntimeABIVersion,
			Dialects: []string{"postgres"}, Clients: []string{"github.com/jackc/pgx/v5@v5.10.0"},
			Capabilities: []Capability{
				CapExplicitBatchBinding, CapExactKeyReadSynthesis, CapOrderedResultMapping,
				CapSparseResultMapping, CapTransactionPartition, CapChunking,
			},
			Notes: "Typed pgx v5 array provider with caller-owned connection/transaction identity and ordinal mapping.",
		},
		{
			SchemaVersion: SchemaVersion, AdapterID: "redis", Version: 1,
			DisplayName: "redis", Category: CategoryBackend, Status: StatusReady, RuntimeABI: RuntimeABIVersion,
			Clients: []string{"github.com/redis/go-redis/v9@v9.21.0"},
			Capabilities: []Capability{
				CapExplicitBatchBinding, CapMGet, CapHMGet, CapPipeline,
				CapClusterSlotPartition, CapOrderedResultMapping,
			},
			Notes: "Typed MGET, HMGET, and explicit pipeline providers with cluster-slot partitioning.",
		},
		{
			SchemaVersion: SchemaVersion, AdapterID: "http/openapi", Version: 1,
			DisplayName: "http/openapi", Category: CategoryNetwork, Status: StatusReady, RuntimeABI: RuntimeABIVersion,
			Capabilities: []Capability{
				CapHTTPExplicitBatch, CapOpenAPIDiscovery, CapOpenAPIExtension,
				CapHTTPJSONArray, CapHTTPKeyedEnvelope, CapHTTPPositionalEnvelope,
				CapTransportPartitioning, CapBoundedChunking, CapProtocolVerification,
			},
			Notes: "Explicit HTTP batch endpoints over net/http; typed keyed/positional JSON envelopes; OpenAPI x-batchweaver extension binding.",
		},
		{
			SchemaVersion: SchemaVersion, AdapterID: "graphql/gqlgen", Version: 1,
			DisplayName: "graphql/gqlgen", Category: CategoryNetwork, Status: StatusReady, RuntimeABI: RuntimeABIVersion,
			Clients: []string{"github.com/99designs/gqlgen@v0.17.94"},
			Capabilities: []Capability{
				CapGraphQLOperationScope, CapGraphQLResolverWave, CapGraphQLSelectionNorm,
				CapGraphQLErrorPath, CapGraphQLNullability, CapGraphQLSubscription,
			},
			Notes: "Public gqlgen extension hooks establish operation scopes and normalized field partitions.",
		},
		{
			SchemaVersion: SchemaVersion, AdapterID: "grpc-go", Version: 1,
			DisplayName: "grpc-go", Category: CategoryNetwork, Status: StatusReady, RuntimeABI: RuntimeABIVersion,
			Clients: []string{"google.golang.org/grpc@v1.83.0"},
			Capabilities: []Capability{
				CapGRPCUnaryBatch, CapGRPCMetadata, CapGRPCStatusDetails, CapGRPCInterceptor,
				CapBoundedChunking, CapTransportPartitioning,
			},
			Notes: "Typed explicit unary batch provider, identity-safe metadata partitions, and bufconn integration coverage.",
		},
	}
	for i := range ms {
		ms[i].Digest = computeDigest(ms[i])
	}
	return ms
}

// Manifests returns the built-in adapter manifests in stable ID order.
func Manifests() []Manifest {
	ms := builtinManifests()
	sort.Slice(ms, func(i, j int) bool { return ms[i].AdapterID < ms[j].AdapterID })
	return ms
}

// ManifestByID returns the manifest for an adapter ID.
func ManifestByID(id string) (Manifest, bool) {
	for _, m := range builtinManifests() {
		if m.AdapterID == id {
			return m, true
		}
	}
	return Manifest{}, false
}
