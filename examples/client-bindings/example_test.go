package clientbindings_test

import (
	"context"
	"testing"

	"github.com/99designs/gqlgen/graphql"
	"github.com/jackc/pgx/v5"
	"github.com/jackc/pgx/v5/pgxpool"
	redis "github.com/redis/go-redis/v9"
	"google.golang.org/grpc"

	batchgqlgen "github.com/Voskan/BatchWeaver/adapters/gqlgen"
	batchgrpc "github.com/Voskan/BatchWeaver/adapters/grpcgo"
	"github.com/Voskan/BatchWeaver/adapters/pgxv5"
	"github.com/Voskan/BatchWeaver/adapters/redisv9"
	batchruntime "github.com/Voskan/BatchWeaver/runtime"
)

var (
	_ pgxv5.Queryer                = (*pgx.Conn)(nil)
	_ pgxv5.Queryer                = (pgx.Tx)(nil)
	_ pgxv5.Queryer                = (*pgxpool.Pool)(nil)
	_ redisv9.MultiGetClient       = (*redis.Client)(nil)
	_ redisv9.PipelineClient       = (*redis.Client)(nil)
	_ graphql.HandlerExtension     = batchgqlgen.ScopeExtension{}
	_ graphql.OperationInterceptor = batchgqlgen.ScopeExtension{}
	_ graphql.FieldInterceptor     = batchgqlgen.ScopeExtension{}
	_ grpc.ClientConnInterface     = (*grpc.ClientConn)(nil)
)

func TestConcreteBindingSurface(t *testing.T) {
	engine, err := batchruntime.NewEngine()
	if err != nil {
		t.Fatal(err)
	}
	defer func() { _ = engine.Close(context.Background()) }()
	if err := (batchgqlgen.ScopeExtension{Engine: engine}).Validate(nil); err != nil {
		t.Fatal(err)
	}
	partitionA, err := batchgrpc.MetadataPartition(context.Background(), nil)
	if err != nil {
		t.Fatal(err)
	}
	partitionB, err := batchgrpc.MetadataPartition(context.Background(), nil)
	if err != nil {
		t.Fatal(err)
	}
	if partitionA != partitionB {
		t.Fatalf("empty metadata partition is nondeterministic: %q != %q", partitionA, partitionB)
	}
}
