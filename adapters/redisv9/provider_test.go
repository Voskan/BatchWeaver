package redisv9_test

import (
	"context"
	"strconv"
	"testing"

	"github.com/alicebob/miniredis/v2"
	redis "github.com/redis/go-redis/v9"

	batchweaver "github.com/Voskan/BatchWeaver"
	"github.com/Voskan/BatchWeaver/adapters/redisv9"
)

func testClient(t *testing.T) (*miniredis.Miniredis, *redis.Client) {
	t.Helper()
	server := miniredis.RunT(t)
	client := redis.NewClient(&redis.Options{Addr: server.Addr()})
	t.Cleanup(func() { _ = client.Close() })
	return server, client
}

func TestMGetProviderMapsSlotsDuplicatesAndMissing(t *testing.T) {
	server, client := testClient(t)
	if err := server.Set("user:{1}:name", "Ada"); err != nil {
		t.Fatal(err)
	}
	if err := server.Set("user:{2}:name", "Lin"); err != nil {
		t.Fatal(err)
	}
	provider := redisv9.MGetProvider[string]{
		Client: client,
		Decode: func(_ string, raw any) (string, error) { return raw.(string), nil },
	}
	req := batchweaver.MustNewBatchRequest([]batchweaver.BatchItem[string]{
		batchweaver.NewBatchItem(1, "user:{2}:name"),
		batchweaver.NewBatchItem(2, "missing:{1}"),
		batchweaver.NewBatchItem(3, "user:{2}:name"),
		batchweaver.NewBatchItem(4, "user:{1}:name"),
	})
	resp, err := provider.Execute(context.Background(), req)
	if err != nil {
		t.Fatal(err)
	}
	out := resp.Outcomes()
	if out[0].Value != "Lin" || !out[1].IsNotFound() || out[2].Value != "Lin" || out[3].Value != "Ada" {
		t.Fatalf("unexpected outcomes: %+v", out)
	}
}

func TestHMGetProviderGroupsHashes(t *testing.T) {
	server, client := testClient(t)
	server.HSet("users", "7", "Ada")
	server.HSet("teams", "3", "Compiler")
	provider := redisv9.HMGetProvider[string]{
		Client: client,
		Decode: func(_ redisv9.HashField, raw any) (string, error) { return raw.(string), nil },
	}
	req := batchweaver.MustNewBatchRequest([]batchweaver.BatchItem[redisv9.HashField]{
		batchweaver.NewBatchItem(1, redisv9.HashField{Hash: "teams", Field: "3"}),
		batchweaver.NewBatchItem(2, redisv9.HashField{Hash: "users", Field: "9"}),
		batchweaver.NewBatchItem(3, redisv9.HashField{Hash: "users", Field: "7"}),
	})
	resp, err := provider.Execute(context.Background(), req)
	if err != nil {
		t.Fatal(err)
	}
	out := resp.Outcomes()
	if out[0].Value != "Compiler" || !out[1].IsNotFound() || out[2].Value != "Ada" {
		t.Fatalf("unexpected outcomes: %+v", out)
	}
}

func TestPipelineProviderPreservesPerItemErrors(t *testing.T) {
	server, client := testClient(t)
	if err := server.Set("count:1", "41"); err != nil {
		t.Fatal(err)
	}
	provider := redisv9.PipelineProvider[string, int]{
		Client: client,
		Queue:  func(ctx context.Context, pipe redis.Pipeliner, key string) redis.Cmder { return pipe.Get(ctx, key) },
		Decode: func(_ string, command redis.Cmder) (int, error) {
			value, err := command.(*redis.StringCmd).Result()
			if err != nil {
				return 0, err
			}
			return strconv.Atoi(value)
		},
	}
	req := batchweaver.MustNewBatchRequest([]batchweaver.BatchItem[string]{
		batchweaver.NewBatchItem(1, "count:1"),
		batchweaver.NewBatchItem(2, "count:missing"),
	})
	resp, err := provider.Execute(context.Background(), req)
	if err != nil {
		t.Fatal(err)
	}
	out := resp.Outcomes()
	if out[0].Value != 41 || !out[1].IsNotFound() {
		t.Fatalf("unexpected outcomes: %+v", out)
	}
}

func TestPipelineProviderKeepsCommandErrorPerItem(t *testing.T) {
	server, client := testClient(t)
	server.HSet("wrong-type", "field", "value")
	if err := server.Set("ok", "value"); err != nil {
		t.Fatal(err)
	}
	provider := redisv9.PipelineProvider[string, string]{
		Client: client,
		Queue:  func(ctx context.Context, pipe redis.Pipeliner, key string) redis.Cmder { return pipe.Get(ctx, key) },
		Decode: func(_ string, command redis.Cmder) (string, error) { return command.(*redis.StringCmd).Result() },
	}
	req := batchweaver.MustNewBatchRequest([]batchweaver.BatchItem[string]{
		batchweaver.NewBatchItem(1, "wrong-type"),
		batchweaver.NewBatchItem(2, "ok"),
	})
	resp, err := provider.Execute(context.Background(), req)
	if err != nil {
		t.Fatalf("per-command error became global: %v", err)
	}
	out := resp.Outcomes()
	if !out[0].IsFailure() || out[1].Value != "value" {
		t.Fatalf("unexpected outcomes: %+v", out)
	}
}
