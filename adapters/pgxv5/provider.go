package pgxv5

import (
	"context"
	"errors"
	"fmt"

	"github.com/jackc/pgx/v5"

	batchweaver "github.com/Voskan/BatchWeaver"
)

// ErrMisconfigured is returned when a provider lacks a required dependency or
// callback.
var ErrMisconfigured = errors.New("batchweaver pgxv5: provider is misconfigured")

// Queryer is implemented by *pgx.Conn, pgx.Tx, and *pgxpool.Pool. Supplying the
// exact caller-owned value preserves the selected transaction or session.
type Queryer interface {
	Query(context.Context, string, ...any) (pgx.Rows, error)
}

// RowScanner is the stable subset of pgx.Rows exposed to decoders.
type RowScanner interface {
	Scan(...any) error
}

// RowDecoder maps the current pgx row to its 1-based request ordinal and typed
// value. Synthesized queries put the ordinal in their first result column.
type RowDecoder[V any] func(RowScanner) (ordinal int, value V, err error)

// ArrayProvider executes one parameterized PostgreSQL query per deterministic
// chunk. Query should accept one array-like argument containing the ordered
// keys and return a 1-based ordinal with each row.
type ArrayProvider[K any, V any] struct {
	Queryer  Queryer
	Query    string
	KeyArg   func([]K) any
	Decode   RowDecoder[V]
	Missing  func(K) error
	MaxItems int
}

// Execute implements the BatchWeaver runtime provider contract. It preserves
// input order and duplicates by correlating results with request ordinals.
func (p ArrayProvider[K, V]) Execute(ctx context.Context, req batchweaver.BatchRequest[K]) (batchweaver.BatchResponse[V], error) {
	if p.Queryer == nil || p.Query == "" || p.Decode == nil {
		return batchweaver.BatchResponse[V]{}, ErrMisconfigured
	}
	items := req.Items()
	maxItems := p.MaxItems
	if maxItems <= 0 {
		maxItems = 1000
	}
	outcomes := make([]batchweaver.Outcome[V], 0, len(items))
	for start := 0; start < len(items); start += maxItems {
		end := min(start+maxItems, len(items))
		chunk, err := p.executeChunk(ctx, items[start:end])
		if err != nil {
			return batchweaver.BatchResponse[V]{}, err
		}
		outcomes = append(outcomes, chunk...)
	}
	return batchweaver.NewBatchResponse(outcomes)
}

func (p ArrayProvider[K, V]) executeChunk(ctx context.Context, items []batchweaver.BatchItem[K]) ([]batchweaver.Outcome[V], error) {
	keys := make([]K, len(items))
	for i, item := range items {
		keys[i] = item.Key
	}
	arg := any(keys)
	if p.KeyArg != nil {
		arg = p.KeyArg(keys)
	}
	rows, err := p.Queryer.Query(ctx, p.Query, arg)
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	values := make(map[int]V, len(items))
	for rows.Next() {
		ordinal, value, decodeErr := p.Decode(rows)
		if decodeErr != nil {
			return nil, fmt.Errorf("batchweaver pgxv5: decode row: %w", decodeErr)
		}
		if ordinal < 1 || ordinal > len(items) {
			return nil, fmt.Errorf("batchweaver pgxv5: response ordinal %d is outside [1,%d]", ordinal, len(items))
		}
		if _, duplicate := values[ordinal]; duplicate {
			return nil, fmt.Errorf("batchweaver pgxv5: duplicate response ordinal %d", ordinal)
		}
		values[ordinal] = value
	}
	if err := rows.Err(); err != nil {
		return nil, err
	}

	out := make([]batchweaver.Outcome[V], len(items))
	for i, item := range items {
		value, ok := values[i+1]
		if ok {
			out[i] = batchweaver.Success(item.ID, value)
			continue
		}
		if p.Missing == nil {
			out[i] = batchweaver.NotFound[V](item.ID)
			continue
		}
		missingErr := p.Missing(item.Key)
		if missingErr == nil {
			out[i] = batchweaver.NotFound[V](item.ID)
		} else {
			out[i] = batchweaver.Failure[V](item.ID, missingErr)
		}
	}
	return out, nil
}
