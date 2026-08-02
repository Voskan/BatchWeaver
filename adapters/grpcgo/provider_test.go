package grpcgo_test

import (
	"context"
	"net"
	"testing"
	"time"

	statuspb "google.golang.org/genproto/googleapis/rpc/status"
	"google.golang.org/grpc"
	"google.golang.org/grpc/codes"
	"google.golang.org/grpc/credentials/insecure"
	"google.golang.org/grpc/metadata"
	"google.golang.org/grpc/status"
	"google.golang.org/grpc/test/bufconn"
	"google.golang.org/protobuf/types/known/structpb"

	batchweaver "github.com/Voskan/BatchWeaver"
	batchgrpc "github.com/Voskan/BatchWeaver/adapters/grpcgo"
)

const batchMethod = "/batchweaver.test.Batch/GetMany"

type batchService interface{ batchServiceMarker() }
type testBatchService struct{}

func (*testBatchService) batchServiceMarker() {}

func batchHandler(server any, ctx context.Context, decode func(any) error, interceptor grpc.UnaryServerInterceptor) (any, error) {
	request := new(structpb.ListValue)
	if err := decode(request); err != nil {
		return nil, err
	}
	handler := func(ctx context.Context, raw any) (any, error) {
		input := raw.(*structpb.ListValue)
		if len(input.Values) > 0 && input.Values[0].GetStringValue() == "block" {
			<-ctx.Done()
			return nil, status.FromContextError(ctx.Err()).Err()
		}
		values := make([]*structpb.Value, len(input.Values))
		for i, value := range input.Values {
			values[i] = structpb.NewStringValue("user:" + value.GetStringValue())
		}
		return &structpb.ListValue{Values: values}, nil
	}
	if interceptor == nil {
		return handler(ctx, request)
	}
	return interceptor(ctx, request, &grpc.UnaryServerInfo{Server: server, FullMethod: batchMethod}, handler)
}

func testProvider(conn grpc.ClientConnInterface) batchgrpc.UnaryProvider[string, string, *structpb.ListValue, *structpb.ListValue] {
	return batchgrpc.UnaryProvider[string, string, *structpb.ListValue, *structpb.ListValue]{
		Conn:   conn,
		Method: batchMethod,
		Build: func(keys []string) (*structpb.ListValue, error) {
			values := make([]*structpb.Value, len(keys))
			for i, key := range keys {
				values[i] = structpb.NewStringValue(key)
			}
			return &structpb.ListValue{Values: values}, nil
		},
		NewResponse: func() *structpb.ListValue { return new(structpb.ListValue) },
		Decode: func(response *structpb.ListValue, items []batchweaver.BatchItem[string]) ([]batchweaver.Outcome[string], error) {
			out := make([]batchweaver.Outcome[string], len(items))
			for i, item := range items {
				out[i] = batchweaver.Success(item.ID, response.Values[i].GetStringValue())
			}
			return out, nil
		},
	}
}

func bufconnClient(t *testing.T) *grpc.ClientConn {
	t.Helper()
	listener := bufconn.Listen(1 << 20)
	server := grpc.NewServer()
	server.RegisterService(&grpc.ServiceDesc{
		ServiceName: "batchweaver.test.Batch",
		HandlerType: (*batchService)(nil),
		Methods:     []grpc.MethodDesc{{MethodName: "GetMany", Handler: batchHandler}},
	}, &testBatchService{})
	go func() { _ = server.Serve(listener) }()
	t.Cleanup(server.Stop)
	conn, err := grpc.NewClient("passthrough:///bufnet",
		grpc.WithTransportCredentials(insecure.NewCredentials()),
		grpc.WithContextDialer(func(context.Context, string) (net.Conn, error) { return listener.Dial() }),
	)
	if err != nil {
		t.Fatal(err)
	}
	t.Cleanup(func() { _ = conn.Close() })
	return conn
}

func TestUnaryProviderUsesExplicitBufconnBatchRPC(t *testing.T) {
	conn := bufconnClient(t)
	provider := testProvider(conn)
	req := batchweaver.MustNewBatchRequest([]batchweaver.BatchItem[string]{
		batchweaver.NewBatchItem(1, "7"), batchweaver.NewBatchItem(2, "9"),
	})
	resp, err := provider.Execute(context.Background(), req)
	if err != nil {
		t.Fatal(err)
	}
	out := resp.Outcomes()
	if out[0].Value != "user:7" || out[1].Value != "user:9" {
		t.Fatalf("unexpected outcomes: %+v", out)
	}
}

func TestUnaryProviderPropagatesCancellation(t *testing.T) {
	ctx, cancel := context.WithTimeout(context.Background(), 25*time.Millisecond)
	defer cancel()
	req := batchweaver.MustNewBatchRequest([]batchweaver.BatchItem[string]{batchweaver.NewBatchItem(1, "block")})
	_, err := testProvider(bufconnClient(t)).Execute(ctx, req)
	if status.Code(err) != codes.DeadlineExceeded {
		t.Fatalf("cancellation code = %v, want %v (err %v)", status.Code(err), codes.DeadlineExceeded, err)
	}
}

func TestStatusOutcomePreservesPerItemGRPCStatus(t *testing.T) {
	failure := batchgrpc.StatusOutcome[string](1, "", false, &statuspb.Status{Code: int32(codes.PermissionDenied), Message: "tenant denied"})
	if !failure.IsFailure() || status.Code(failure.Err) != codes.PermissionDenied {
		t.Fatalf("unexpected failure outcome: %+v", failure)
	}
	if got := batchgrpc.StatusOutcome(2, "value", true, nil); !got.IsSuccess() || got.Value != "value" {
		t.Fatalf("unexpected success outcome: %+v", got)
	}
	if got := batchgrpc.StatusOutcome[string](3, "", false, &statuspb.Status{}); !got.IsNotFound() {
		t.Fatalf("unexpected missing outcome: %+v", got)
	}
}

func TestMetadataPartitionIsIdentitySensitiveAndTraceInsensitive(t *testing.T) {
	ctxA := metadata.NewOutgoingContext(context.Background(), metadata.Pairs("authorization", "Bearer A", "traceparent", "trace-a"))
	ctxB := metadata.NewOutgoingContext(context.Background(), metadata.Pairs("authorization", "Bearer B", "traceparent", "trace-b"))
	partitionA, err := batchgrpc.MetadataPartition(ctxA, nil)
	if err != nil {
		t.Fatal(err)
	}
	partitionB, err := batchgrpc.MetadataPartition(ctxB, nil)
	if err != nil {
		t.Fatal(err)
	}
	if partitionA == partitionB {
		t.Fatal("different authorization metadata shared a partition")
	}
	traceA, err := batchgrpc.MetadataPartition(metadata.NewOutgoingContext(context.Background(), metadata.Pairs("traceparent", "a")), nil)
	if err != nil {
		t.Fatal(err)
	}
	traceB, err := batchgrpc.MetadataPartition(metadata.NewOutgoingContext(context.Background(), metadata.Pairs("traceparent", "b")), nil)
	if err != nil {
		t.Fatal(err)
	}
	if traceA != traceB {
		t.Fatal("trace-only metadata changed batching partition")
	}
}

func TestMetadataPartitionRejectsForbiddenKey(t *testing.T) {
	ctx := metadata.NewOutgoingContext(context.Background(), metadata.Pairs("x-secret", "value"))
	if _, err := batchgrpc.MetadataPartition(ctx, map[string]batchgrpc.MetadataPolicy{"x-secret": batchgrpc.MetadataForbidden}); err == nil {
		t.Fatal("forbidden metadata accepted")
	}
}

func TestUnaryProviderRejectsNilGeneratedMessage(t *testing.T) {
	provider := batchgrpc.UnaryProvider[string, string, *structpb.ListValue, *structpb.ListValue]{
		Conn:        bufconnClient(t),
		Method:      batchMethod,
		Build:       func([]string) (*structpb.ListValue, error) { return nil, nil },
		NewResponse: func() *structpb.ListValue { return new(structpb.ListValue) },
		Decode: func(*structpb.ListValue, []batchweaver.BatchItem[string]) ([]batchweaver.Outcome[string], error) {
			return nil, nil
		},
	}
	req := batchweaver.MustNewBatchRequest([]batchweaver.BatchItem[string]{batchweaver.NewBatchItem(1, "7")})
	if _, err := provider.Execute(context.Background(), req); err == nil {
		t.Fatal("nil generated request accepted")
	}
}
