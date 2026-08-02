package redisv9

import (
	"context"
	"errors"
	"fmt"
	"io"
	"net"
	"sort"

	redis "github.com/redis/go-redis/v9"

	batchweaver "github.com/Voskan/BatchWeaver"
	internaladapter "github.com/Voskan/BatchWeaver/internal/adapter"
)

// ErrMisconfigured is returned when a provider lacks a client or callback.
var ErrMisconfigured = errors.New("batchweaver redisv9: provider is misconfigured")

// MultiGetClient is implemented by go-redis clients that support MGET and
// HMGET, including *redis.Client and *redis.ClusterClient.
type MultiGetClient interface {
	MGet(context.Context, ...string) *redis.SliceCmd
	HMGet(context.Context, string, ...string) *redis.SliceCmd
}

// Decoder converts a non-nil Redis value to V. Missing values bypass Decoder.
type Decoder[V any] func(key string, value any) (V, error)

// MGetProvider executes MGET calls without crossing Redis Cluster hash slots.
type MGetProvider[V any] struct {
	Client   MultiGetClient
	Decode   Decoder[V]
	Missing  func(key string) error
	MaxItems int
}

// Execute implements the BatchWeaver runtime provider contract.
func (p MGetProvider[V]) Execute(ctx context.Context, req batchweaver.BatchRequest[string]) (batchweaver.BatchResponse[V], error) {
	if p.Client == nil || p.Decode == nil {
		return batchweaver.BatchResponse[V]{}, ErrMisconfigured
	}
	items := req.Items()
	keys := make([]string, len(items))
	for i, item := range items {
		keys[i] = item.Key
	}
	out := make([]batchweaver.Outcome[V], len(items))
	maxItems := p.MaxItems
	if maxItems <= 0 {
		maxItems = 1000
	}
	for _, group := range internaladapter.SlotGroups(keys) {
		for start := 0; start < len(group.Keys); start += maxItems {
			end := min(start+maxItems, len(group.Keys))
			values, err := p.Client.MGet(ctx, group.Keys[start:end]...).Result()
			if err != nil {
				return batchweaver.BatchResponse[V]{}, err
			}
			if len(values) != end-start {
				return batchweaver.BatchResponse[V]{}, fmt.Errorf("batchweaver redisv9: MGET returned %d values for %d keys", len(values), end-start)
			}
			for i, raw := range values {
				original := group.Indices[start+i]
				out[original] = p.outcome(items[original], raw)
			}
		}
	}
	return batchweaver.NewBatchResponse(out)
}

func (p MGetProvider[V]) outcome(item batchweaver.BatchItem[string], raw any) batchweaver.Outcome[V] {
	if raw == nil {
		if p.Missing == nil {
			return batchweaver.NotFound[V](item.ID)
		}
		if err := p.Missing(item.Key); err != nil {
			return batchweaver.Failure[V](item.ID, err)
		}
		return batchweaver.NotFound[V](item.ID)
	}
	value, err := p.Decode(item.Key, raw)
	if err != nil {
		return batchweaver.Failure[V](item.ID, err)
	}
	return batchweaver.Success(item.ID, value)
}

// HashField identifies one field inside a Redis hash.
type HashField struct {
	Hash  string
	Field string
}

// HMGetProvider groups fields by hash key and executes one HMGET per group.
type HMGetProvider[V any] struct {
	Client   MultiGetClient
	Decode   func(HashField, any) (V, error)
	Missing  func(HashField) error
	MaxItems int
}

// Execute implements the BatchWeaver runtime provider contract.
func (p HMGetProvider[V]) Execute(ctx context.Context, req batchweaver.BatchRequest[HashField]) (batchweaver.BatchResponse[V], error) {
	if p.Client == nil || p.Decode == nil {
		return batchweaver.BatchResponse[V]{}, ErrMisconfigured
	}
	items := req.Items()
	byHash := make(map[string][]int)
	var hashes []string
	for i, item := range items {
		if _, ok := byHash[item.Key.Hash]; !ok {
			hashes = append(hashes, item.Key.Hash)
		}
		byHash[item.Key.Hash] = append(byHash[item.Key.Hash], i)
	}
	sort.Strings(hashes)
	out := make([]batchweaver.Outcome[V], len(items))
	maxItems := p.MaxItems
	if maxItems <= 0 {
		maxItems = 1000
	}
	for _, hash := range hashes {
		indices := byHash[hash]
		for start := 0; start < len(indices); start += maxItems {
			end := min(start+maxItems, len(indices))
			fields := make([]string, end-start)
			for i, index := range indices[start:end] {
				fields[i] = items[index].Key.Field
			}
			values, err := p.Client.HMGet(ctx, hash, fields...).Result()
			if err != nil {
				return batchweaver.BatchResponse[V]{}, err
			}
			if len(values) != len(fields) {
				return batchweaver.BatchResponse[V]{}, fmt.Errorf("batchweaver redisv9: HMGET returned %d values for %d fields", len(values), len(fields))
			}
			for i, raw := range values {
				index := indices[start+i]
				item := items[index]
				if raw == nil {
					if p.Missing == nil {
						out[index] = batchweaver.NotFound[V](item.ID)
					} else if missingErr := p.Missing(item.Key); missingErr != nil {
						out[index] = batchweaver.Failure[V](item.ID, missingErr)
					} else {
						out[index] = batchweaver.NotFound[V](item.ID)
					}
					continue
				}
				value, decodeErr := p.Decode(item.Key, raw)
				if decodeErr != nil {
					out[index] = batchweaver.Failure[V](item.ID, decodeErr)
				} else {
					out[index] = batchweaver.Success(item.ID, value)
				}
			}
		}
	}
	return batchweaver.NewBatchResponse(out)
}

// PipelineClient is implemented by go-redis clients that support pipelining.
type PipelineClient interface {
	Pipelined(context.Context, func(redis.Pipeliner) error) ([]redis.Cmder, error)
}

// PipelineProvider pipelines one explicitly supplied command per request item.
// It does not infer commands or merge writes.
type PipelineProvider[K any, V any] struct {
	Client      PipelineClient
	Queue       func(context.Context, redis.Pipeliner, K) redis.Cmder
	Decode      func(K, redis.Cmder) (V, error)
	GlobalError func(error) bool
}

// Execute implements the BatchWeaver runtime provider contract.
func (p PipelineProvider[K, V]) Execute(ctx context.Context, req batchweaver.BatchRequest[K]) (batchweaver.BatchResponse[V], error) {
	if p.Client == nil || p.Queue == nil || p.Decode == nil {
		return batchweaver.BatchResponse[V]{}, ErrMisconfigured
	}
	items := req.Items()
	queued := make([]redis.Cmder, len(items))
	_, err := p.Client.Pipelined(ctx, func(pipe redis.Pipeliner) error {
		for i, item := range items {
			queued[i] = p.Queue(ctx, pipe, item.Key)
			if queued[i] == nil {
				return fmt.Errorf("batchweaver redisv9: Queue returned nil command for item %d", i)
			}
		}
		return nil
	})
	globalError := p.GlobalError
	if globalError == nil {
		globalError = defaultGlobalError
	}
	if err != nil && !errors.Is(err, redis.Nil) && globalError(err) {
		return batchweaver.BatchResponse[V]{}, err
	}
	out := make([]batchweaver.Outcome[V], len(items))
	for i, item := range items {
		value, decodeErr := p.Decode(item.Key, queued[i])
		switch {
		case errors.Is(decodeErr, redis.Nil):
			out[i] = batchweaver.NotFound[V](item.ID)
		case decodeErr != nil:
			out[i] = batchweaver.Failure[V](item.ID, decodeErr)
		default:
			out[i] = batchweaver.Success(item.ID, value)
		}
	}
	return batchweaver.NewBatchResponse(out)
}

func defaultGlobalError(err error) bool {
	if errors.Is(err, context.Canceled) || errors.Is(err, context.DeadlineExceeded) || errors.Is(err, io.EOF) {
		return true
	}
	var networkError net.Error
	return errors.As(err, &networkError)
}
