package adapter

import (
	"crypto/sha256"
	"encoding/hex"
	"fmt"
	"io"
	"sort"
)

// SchemaVersion identifies the adapter manifest schema. It is independent from
// the analysis, proof, transformation, and bridge schema versions.
const SchemaVersion = "batchweaver.adapter/v1alpha1"

// Capability is a member of the closed adapter capability vocabulary. Unknown
// capabilities are rejected.
type Capability string

// The closed capability vocabulary. A capability is only listed on a manifest
// once it is implemented and tested.
const (
	CapExplicitBatchBinding  Capability = "explicit-batch-binding"
	CapExactKeyReadSynthesis Capability = "exact-key-read-synthesis"
	CapCompositeKeyRead      Capability = "composite-key-read-synthesis"
	CapBoundedJoinRead       Capability = "bounded-join-read-synthesis"
	CapOrderedResultMapping  Capability = "ordered-result-mapping"
	CapKeyedResultMapping    Capability = "keyed-result-mapping"
	CapSparseResultMapping   Capability = "sparse-result-mapping"
	CapPerItemError          Capability = "per-item-error"
	CapGlobalError           Capability = "global-error"
	CapTransactionPartition  Capability = "transaction-partitioning"
	CapSessionPartition      Capability = "session-partitioning"
	CapPreparedStatements    Capability = "prepared-statements"
	CapChunking              Capability = "chunking"
	CapPipeline              Capability = "pipeline"
	CapClusterSlotPartition  Capability = "cluster-slot-partitioning"
	CapSemanticVerification  Capability = "semantic-verification"
	CapGeneratedRowDecoding  Capability = "generated-row-decoding"
	CapMGet                  Capability = "mget"
	CapHMGet                 Capability = "hmget"

	// Network protocol capabilities.
	CapGraphQLOperationScope  Capability = "graphql-operation-scope"
	CapGraphQLResolverWave    Capability = "graphql-resolver-wave"
	CapGraphQLSelectionNorm   Capability = "graphql-selection-normalization"
	CapGraphQLErrorPath       Capability = "graphql-error-path"
	CapGraphQLNullability     Capability = "graphql-nullability"
	CapGraphQLSubscription    Capability = "graphql-subscription-awareness"
	CapGRPCUnaryBatch         Capability = "grpc-unary-explicit-batch"
	CapGRPCClientStreaming    Capability = "grpc-client-streaming-explicit"
	CapGRPCServerStreaming    Capability = "grpc-server-streaming-explicit"
	CapGRPCBidiStreaming      Capability = "grpc-bidi-streaming-explicit"
	CapGRPCMetadata           Capability = "grpc-metadata"
	CapGRPCStatusDetails      Capability = "grpc-status-details"
	CapGRPCInterceptor        Capability = "grpc-interceptor-compatible"
	CapHTTPExplicitBatch      Capability = "http-explicit-batch"
	CapOpenAPIDiscovery       Capability = "openapi-discovery"
	CapOpenAPIExtension       Capability = "openapi-extension-binding"
	CapHTTPJSONArray          Capability = "http-json-array"
	CapHTTPKeyedEnvelope      Capability = "http-keyed-envelope"
	CapHTTPPositionalEnvelope Capability = "http-positional-envelope"
	CapHTTPStreamingNDJSON    Capability = "http-streaming-ndjson"
	CapProtocolVerification   Capability = "protocol-contract-verification"
	CapTransportPartitioning  Capability = "transport-partitioning"
	CapBoundedChunking        Capability = "bounded-chunking"
)

// knownCapabilities is the set used to reject unknown capabilities.
var knownCapabilities = map[Capability]struct{}{
	CapExplicitBatchBinding: {}, CapExactKeyReadSynthesis: {}, CapCompositeKeyRead: {}, CapBoundedJoinRead: {},
	CapOrderedResultMapping: {}, CapKeyedResultMapping: {}, CapSparseResultMapping: {},
	CapPerItemError: {}, CapGlobalError: {}, CapTransactionPartition: {}, CapSessionPartition: {},
	CapPreparedStatements: {}, CapChunking: {}, CapPipeline: {}, CapClusterSlotPartition: {},
	CapSemanticVerification: {}, CapGeneratedRowDecoding: {}, CapMGet: {}, CapHMGet: {},
	CapGraphQLOperationScope: {}, CapGraphQLResolverWave: {}, CapGraphQLSelectionNorm: {},
	CapGraphQLErrorPath: {}, CapGraphQLNullability: {}, CapGraphQLSubscription: {},
	CapGRPCUnaryBatch: {}, CapGRPCClientStreaming: {}, CapGRPCServerStreaming: {},
	CapGRPCBidiStreaming: {}, CapGRPCMetadata: {}, CapGRPCStatusDetails: {}, CapGRPCInterceptor: {},
	CapHTTPExplicitBatch: {}, CapOpenAPIDiscovery: {}, CapOpenAPIExtension: {},
	CapHTTPJSONArray: {}, CapHTTPKeyedEnvelope: {}, CapHTTPPositionalEnvelope: {},
	CapHTTPStreamingNDJSON: {}, CapProtocolVerification: {}, CapTransportPartitioning: {},
	CapBoundedChunking: {},
}

// Valid reports whether c is a known capability.
func (c Capability) Valid() bool { _, ok := knownCapabilities[c]; return ok }

// Status is an adapter's implementation status in this build.
type Status string

const (
	// StatusReady means the adapter's declared capabilities are implemented here.
	StatusReady Status = "ready"
	// StatusDeferred means the adapter contract is defined but its concrete
	// client binding is not compiled into this build (see limitations).
	StatusDeferred Status = "deferred"
)

// Manifest is the versioned, deterministic description of an adapter.
type Manifest struct {
	SchemaVersion string       `json:"schema_version"`
	AdapterID     string       `json:"adapter_id"`
	Version       int          `json:"version"`
	DisplayName   string       `json:"display_name"`
	Category      string       `json:"category"`
	Status        Status       `json:"status"`
	RuntimeABI    string       `json:"runtime_abi"`
	Capabilities  []Capability `json:"capabilities"`
	Dialects      []string     `json:"dialects,omitempty"`
	Clients       []string     `json:"clients,omitempty"`
	Notes         string       `json:"notes,omitempty"`
	Digest        string       `json:"digest"`
}

// Validate checks the manifest for a known schema, non-empty identity, and only
// known capabilities.
func (m Manifest) Validate() error {
	if m.SchemaVersion != SchemaVersion {
		return fmt.Errorf("adapter %q: unsupported manifest schema %q", m.AdapterID, m.SchemaVersion)
	}
	if m.AdapterID == "" {
		return fmt.Errorf("adapter manifest has no ID")
	}
	for _, c := range m.Capabilities {
		if !c.Valid() {
			return fmt.Errorf("adapter %q: unknown capability %q", m.AdapterID, c)
		}
	}
	return nil
}

// HasCapability reports whether the manifest declares a capability.
func (m Manifest) HasCapability(c Capability) bool {
	for _, x := range m.Capabilities {
		if x == c {
			return true
		}
	}
	return false
}

// computeDigest returns a deterministic digest of a manifest's semantic content,
// excluding the Digest field itself and any host-specific data.
func computeDigest(m Manifest) string {
	h := sha256.New()
	write := func(s string) { _, _ = io.WriteString(h, s); _, _ = h.Write([]byte{0}) }
	write(m.SchemaVersion)
	write(m.AdapterID)
	write(fmt.Sprintf("v%d", m.Version))
	write(m.Category)
	write(string(m.Status))
	write(m.RuntimeABI)
	caps := make([]string, 0, len(m.Capabilities))
	for _, c := range m.Capabilities {
		caps = append(caps, string(c))
	}
	sort.Strings(caps)
	for _, c := range caps {
		write(c)
	}
	dl := append([]string(nil), m.Dialects...)
	sort.Strings(dl)
	for _, d := range dl {
		write(d)
	}
	cl := append([]string(nil), m.Clients...)
	sort.Strings(cl)
	for _, c := range cl {
		write(c)
	}
	return "sha256:" + hex.EncodeToString(h.Sum(nil))
}
