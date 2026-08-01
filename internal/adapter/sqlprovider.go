package adapter

import (
	"context"
	"database/sql"
	"errors"
	"fmt"

	batchweaver "github.com/Voskan/BatchWeaver"
)

// Queryer is the subset of *sql.DB, *sql.Tx, and *sql.Conn used by the SQL
// provider. Using an interface preserves the caller's transaction and connection
// identity: the provider never acquires its own connection or transaction.
type Queryer interface {
	QueryContext(ctx context.Context, query string, args ...any) (*sql.Rows, error)
}

// SQLProvider executes a synthesized exact-key batch query and maps rows to
// ordered scalar outcomes. It is generic over the key and value types and uses
// no reflection: the caller supplies a typed key-array builder and row decoder.
type SQLProvider[K any, V any] struct {
	// DB is the caller-owned queryer (a *sql.DB, *sql.Tx, or *sql.Conn), preserving
	// transaction and connection identity.
	DB Queryer
	// Plan is the synthesized query plan.
	Plan SynthPlan
	// KeyArg builds the single array parameter ($1) from the ordered keys,
	// wrapping them in a driver-appropriate array value.
	KeyArg func([]K) any
	// Decode reads one result row and returns its 1-based ordinal within the batch
	// and the decoded value. It must scan the leading bw_ord column first, then the
	// projection columns.
	Decode func(*sql.Rows) (int64, V, error)
	// Missing returns the outcome error for a requested key with no row; it
	// defaults to sql.ErrNoRows.
	Missing func() error
	// Limits bounds one batch; larger batches are chunked deterministically.
	Limits ParameterLimits
}

// Execute runs the batch query (chunked as needed) and returns one outcome per
// request item in order, preserving duplicates and the declared missing outcome.
// A query error is a global failure for the affected chunk's items.
func (p SQLProvider[K, V]) Execute(ctx context.Context, req batchweaver.BatchRequest[K]) (batchweaver.BatchResponse[V], error) {
	items := req.Items()
	n := len(items)
	limits := p.Limits
	if limits.MaxItems <= 0 {
		limits = DefaultLimits()
	}
	missing := p.Missing
	if missing == nil {
		missing = func() error { return sql.ErrNoRows }
	}

	outcomes := make([]batchweaver.Outcome[V], 0, n)
	for _, rng := range Chunks(n, limits) {
		chunk := items[rng[0]:rng[1]]
		chunkOutcomes, err := p.executeChunk(ctx, chunk, missing)
		if err != nil {
			return batchweaver.BatchResponse[V]{}, err
		}
		outcomes = append(outcomes, chunkOutcomes...)
	}
	return batchweaver.NewBatchResponse(outcomes)
}

// executeChunk runs one chunk and maps rows to outcomes by ordinal.
func (p SQLProvider[K, V]) executeChunk(ctx context.Context, chunk []batchweaver.BatchItem[K], missing func() error) ([]batchweaver.Outcome[V], error) {
	keys := make([]K, len(chunk))
	for i, it := range chunk {
		keys[i] = it.Key
	}
	arg := any(keys)
	if p.KeyArg != nil {
		arg = p.KeyArg(keys)
	}
	rows, err := p.DB.QueryContext(ctx, p.Plan.Query, arg)
	if err != nil {
		return nil, err
	}
	defer func() { _ = rows.Close() }()

	values := make(map[int64]V, len(chunk))
	for rows.Next() {
		ord, v, derr := p.Decode(rows)
		if derr != nil {
			return nil, fmt.Errorf("row decode: %w", derr)
		}
		if ord >= 1 && int(ord) <= len(chunk) {
			values[ord] = v
		}
	}
	if err := rows.Err(); err != nil {
		return nil, err
	}

	out := make([]batchweaver.Outcome[V], len(chunk))
	for i, it := range chunk {
		if v, ok := values[int64(i+1)]; ok {
			out[i] = batchweaver.Success(it.ID, v)
			continue
		}
		out[i] = batchweaver.Failure[V](it.ID, missing())
	}
	return out, nil
}

// ErrProviderMisconfigured is returned when a provider lacks a required field.
var ErrProviderMisconfigured = errors.New("adapter: SQL provider is misconfigured")
