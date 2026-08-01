package adapter

import (
	"fmt"
	"strings"
)

// gRPC diagnostic codes (BW72xx).
const (
	CodeGRPCBatchMethodMissing   = "BW7201"
	CodeGRPCKeyAmbiguous         = "BW7202"
	CodeGRPCMetadataIncompatible = "BW7203"
	CodeGRPCCallOption           = "BW7204"
	CodeGRPCStatusIncomplete     = "BW7205"
	CodeGRPCStreamCorrelation    = "BW7206"
	CodeGRPCInterceptorUnknown   = "BW7207"
	CodeGRPCMessageLimit         = "BW7208"
)

// GRPCResponseMode selects how a batch response correlates to requests.
type GRPCResponseMode string

// gRPC batch response correlation modes.
const (
	GRPCKeyed      GRPCResponseMode = "keyed"
	GRPCPositional GRPCResponseMode = "positional"
)

// GRPCBinding is an explicit unary scalar-to-batch RPC binding. It is never
// inferred; it must be declared in configuration and validated strictly.
type GRPCBinding struct {
	ScalarMethod       string // e.g. /users.v1.UserService/GetUser
	BatchMethod        string // e.g. /users.v1.UserService/BatchGetUsers
	RequestKey         string // request message field carrying the key
	BatchRequestsField string // repeated field in the batch request
	ResponseMode       GRPCResponseMode
	ResponseKey        string // response item field carrying the correlation key (keyed mode)
	PerItemStatusField string // optional per-item google.rpc.Status field
	MaxItems           int
}

// Validate checks a gRPC binding for completeness.
func (b GRPCBinding) Validate() *Rejection {
	if b.ScalarMethod == "" || b.BatchMethod == "" {
		return &Rejection{Code: CodeGRPCBatchMethodMissing, Reason: "scalar and batch methods are required"}
	}
	if b.RequestKey == "" || b.BatchRequestsField == "" {
		return &Rejection{Code: CodeGRPCKeyAmbiguous, Reason: "request key and batch requests field are required"}
	}
	switch b.ResponseMode {
	case GRPCKeyed:
		if b.ResponseKey == "" {
			return &Rejection{Code: CodeGRPCKeyAmbiguous, Reason: "keyed response mode requires a response key field"}
		}
	case GRPCPositional:
	default:
		return &Rejection{Code: CodeGRPCKeyAmbiguous, Reason: "response mode must be keyed or positional"}
	}
	return nil
}

// MetadataPolicy classifies how a metadata key may be treated when coalescing.
type MetadataPolicy string

const (
	// MetaMustEqual means callers may batch only when the value is identical.
	MetaMustEqual MetadataPolicy = "must-equal"
	// MetaPartition means differing values force separate batches.
	MetaPartition MetadataPolicy = "partition"
	// MetaMerge means values may be combined at the batch level (e.g. tracing).
	MetaMerge MetadataPolicy = "merge"
	// MetaForbidden means the key must never appear on a merged batch request.
	MetaForbidden MetadataPolicy = "forbidden"
)

// ClassifyMetadata returns the default policy for a gRPC metadata key. gRPC-Go
// lowercases metadata keys; classification is case-insensitive. Security- and
// routing-sensitive keys partition (so different callers never share a batch);
// tracing keys may merge; unknown keys partition conservatively.
func ClassifyMetadata(key string) MetadataPolicy {
	k := strings.ToLower(key)
	switch {
	case k == "authorization" || strings.Contains(k, "token") || strings.Contains(k, "api-key") ||
		strings.Contains(k, "cookie") || strings.HasSuffix(k, "-bin") && strings.Contains(k, "auth"):
		return MetaPartition
	case strings.HasPrefix(k, "x-tenant") || strings.HasPrefix(k, "x-authorization") ||
		strings.HasPrefix(k, "x-region") || strings.HasPrefix(k, "x-consistency"):
		return MetaPartition
	case k == "traceparent" || k == "tracestate" || strings.HasPrefix(k, "x-b3-") || k == "grpc-trace-bin":
		return MetaMerge
	case k == "content-type" || k == "grpc-accept-encoding" || k == "grpc-encoding":
		return MetaMustEqual
	default:
		return MetaPartition
	}
}

// StreamingMode names a supported gRPC streaming shape (contracts only in this
// stage).
type StreamingMode string

// Supported gRPC streaming shapes (contracts only in this stage).
const (
	StreamUnary           StreamingMode = "unary"
	StreamClientStreaming StreamingMode = "client-streaming"
	StreamServerStreaming StreamingMode = "server-streaming"
	StreamBidi            StreamingMode = "bidi"
)

// StreamState is a state in the explicit streaming lifecycle state machine.
type StreamState string

// Streaming lifecycle states.
const (
	StreamOpening    StreamState = "opening"
	StreamReady      StreamState = "ready"
	StreamHalfClosed StreamState = "half-closed"
	StreamDraining   StreamState = "draining"
	StreamFailed     StreamState = "failed"
	StreamClosed     StreamState = "closed"
)

// String renders a binding summary for inspection.
func (b GRPCBinding) String() string {
	return fmt.Sprintf("scalar=%s batch=%s key=%s mode=%s", b.ScalarMethod, b.BatchMethod, b.RequestKey, b.ResponseMode)
}
