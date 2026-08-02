package grpcgo

import (
	"context"
	"crypto/sha256"
	"encoding/hex"
	"errors"
	"fmt"
	"reflect"
	"sort"
	"strings"

	statuspb "google.golang.org/genproto/googleapis/rpc/status"
	"google.golang.org/grpc"
	"google.golang.org/grpc/codes"
	"google.golang.org/grpc/metadata"
	"google.golang.org/grpc/status"

	batchweaver "github.com/Voskan/BatchWeaver"
)

// ErrMisconfigured is returned when an explicit batch binding is incomplete.
var ErrMisconfigured = errors.New("batchweaver grpcgo: provider is misconfigured")

// ErrForbiddenMetadata is returned when metadata prohibited by policy is
// present on an outgoing call.
var ErrForbiddenMetadata = errors.New("batchweaver grpcgo: forbidden metadata")

// UnaryProvider invokes one explicit batch RPC for each runtime batch.
// BatchRequest and BatchResponse are normally generated protobuf pointer types.
type UnaryProvider[K any, V any, BatchRequest any, BatchResponse any] struct {
	Conn        grpc.ClientConnInterface
	Method      string
	Build       func([]K) (BatchRequest, error)
	NewResponse func() BatchResponse
	Decode      func(BatchResponse, []batchweaver.BatchItem[K]) ([]batchweaver.Outcome[V], error)
	Options     []grpc.CallOption
}

// StatusOutcome converts the conventional google.rpc.Status carried by one
// batch response item into a BatchWeaver outcome. A nil status and OK both use
// found to distinguish success from not-found. Non-OK status codes preserve
// their gRPC code, message, and protobuf details through status.FromProto.
func StatusOutcome[V any](id batchweaver.RequestID, value V, found bool, itemStatus *statuspb.Status) batchweaver.Outcome[V] {
	if itemStatus != nil && codes.Code(itemStatus.Code) != codes.OK {
		return batchweaver.Failure[V](id, status.FromProto(itemStatus).Err())
	}
	if !found {
		return batchweaver.NotFound[V](id)
	}
	return batchweaver.Success(id, value)
}

// Execute implements the BatchWeaver runtime provider contract.
func (p UnaryProvider[K, V, BatchRequest, BatchResponse]) Execute(ctx context.Context, req batchweaver.BatchRequest[K]) (batchweaver.BatchResponse[V], error) {
	if p.Conn == nil || p.Method == "" || p.Build == nil || p.NewResponse == nil || p.Decode == nil {
		return batchweaver.BatchResponse[V]{}, ErrMisconfigured
	}
	items := req.Items()
	keys := make([]K, len(items))
	for i, item := range items {
		keys[i] = item.Key
	}
	batchRequest, err := p.Build(keys)
	if err != nil {
		return batchweaver.BatchResponse[V]{}, fmt.Errorf("batchweaver grpcgo: build batch request: %w", err)
	}
	batchResponse := p.NewResponse()
	if nilValue(batchRequest) || nilValue(batchResponse) {
		return batchweaver.BatchResponse[V]{}, fmt.Errorf("%w: request and response messages must be non-nil", ErrMisconfigured)
	}
	if err := p.Conn.Invoke(ctx, p.Method, batchRequest, batchResponse, p.Options...); err != nil {
		return batchweaver.BatchResponse[V]{}, err
	}
	outcomes, err := p.Decode(batchResponse, items)
	if err != nil {
		return batchweaver.BatchResponse[V]{}, fmt.Errorf("batchweaver grpcgo: decode batch response: %w", err)
	}
	response, err := batchweaver.NewBatchResponse(outcomes)
	if err != nil {
		return batchweaver.BatchResponse[V]{}, err
	}
	if err := response.ValidateAgainst(req.IDs()); err != nil {
		return batchweaver.BatchResponse[V]{}, fmt.Errorf("batchweaver grpcgo: correlate batch response: %w", err)
	}
	return response, nil
}

func nilValue(value any) bool {
	if value == nil {
		return true
	}
	rv := reflect.ValueOf(value)
	switch rv.Kind() {
	case reflect.Chan, reflect.Func, reflect.Interface, reflect.Map, reflect.Pointer, reflect.Slice:
		return rv.IsNil()
	default:
		return false
	}
}

// MetadataPolicy controls how one outgoing metadata key participates in a
// batching partition.
type MetadataPolicy string

const (
	// MetadataPartitionKey includes the key and values in the partition digest.
	MetadataPartitionKey MetadataPolicy = "partition"
	// MetadataMergeable excludes tracing-only metadata from the digest.
	MetadataMergeable MetadataPolicy = "merge"
	// MetadataForbidden rejects calls carrying the key.
	MetadataForbidden MetadataPolicy = "forbidden"
)

// DefaultMetadataPolicy conservatively partitions credentials, tenants,
// routing metadata, binary metadata, and all unknown keys. Trace propagation
// keys are mergeable because the provider call receives its own trace context.
func DefaultMetadataPolicy(key string) MetadataPolicy {
	key = strings.ToLower(key)
	switch {
	case key == "traceparent", key == "tracestate", key == "grpc-trace-bin", strings.HasPrefix(key, "x-b3-"):
		return MetadataMergeable
	default:
		return MetadataPartitionKey
	}
}

// MetadataPartition returns an identity-safe digest of outgoing metadata.
// Overrides are matched case-insensitively. Raw metadata is never returned.
func MetadataPartition(ctx context.Context, overrides map[string]MetadataPolicy) (string, error) {
	md, ok := metadata.FromOutgoingContext(ctx)
	if !ok || len(md) == 0 {
		return "grpc-metadata:none", nil
	}
	policies := make(map[string]MetadataPolicy, len(overrides))
	for key, policy := range overrides {
		policies[strings.ToLower(key)] = policy
	}
	keys := make([]string, 0, len(md))
	for key := range md {
		keys = append(keys, strings.ToLower(key))
	}
	sort.Strings(keys)
	hash := sha256.New()
	for _, key := range keys {
		policy, configured := policies[key]
		if !configured {
			policy = DefaultMetadataPolicy(key)
		}
		switch policy {
		case MetadataMergeable:
			continue
		case MetadataForbidden:
			return "", fmt.Errorf("%w: %s", ErrForbiddenMetadata, key)
		case MetadataPartitionKey:
		default:
			return "", fmt.Errorf("batchweaver grpcgo: invalid metadata policy %q for %s", policy, key)
		}
		_, _ = hash.Write([]byte(key))
		_, _ = hash.Write([]byte{0})
		for _, value := range md.Get(key) {
			_, _ = hash.Write([]byte(value))
			_, _ = hash.Write([]byte{0})
		}
	}
	return "grpc-metadata:sha256:" + hex.EncodeToString(hash.Sum(nil)), nil
}
