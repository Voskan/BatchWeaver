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
	// KeyArgs builds one array argument per composite-key component. Arguments
	// must be returned in PostgreSQL placeholder order ($1, $2, ...). It takes
	// precedence over KeyArg and is required for composite keys.
	KeyArgs func([]K) []any
	// Decode reads one result row and returns its 1-based ordinal within the batch
	// and the decoded value. It must scan the leading bw_ord column first, then the
	// projection columns.
	Decode func(*sql.Rows) (int64, V, error)
	// Missing returns the outcome error for a requested key with no row; it
	// defaults to sql.ErrNoRows.
	Missing func() error
	// Limits bounds one batch; larger batches are chunked deterministically.
	Limits ParameterLimits
	// EstimatePayload optionally returns the encoded key payload size in bytes.
	// When provided, MaxPayloadKB is enforced before any database I/O.
	EstimatePayload func([]K) int
}

// Execute runs the batch query (chunked as needed) and returns one outcome per
// request item in order, preserving duplicates and the declared missing outcome.
// A query error is a global failure for the affected chunk's items.
func (p SQLProvider[K, V]) Execute(ctx context.Context, req batchweaver.BatchRequest[K]) (batchweaver.BatchResponse[V], error) {
	if p.DB == nil || p.Plan.Query == "" || p.Decode == nil {
		return batchweaver.BatchResponse[V]{}, ErrProviderMisconfigured
	}
	if err := p.Plan.Validate(); err != nil {
		return batchweaver.BatchResponse[V]{}, fmt.Errorf("%w: %w", ErrProviderMisconfigured, err)
	}
	items := req.Items()
	n := len(items)
	limits := p.Limits
	if limits == (ParameterLimits{}) {
		limits = p.Plan.Limits
	}
	limits = normalizedLimits(limits)
	missing := p.Missing
	if missing == nil {
		missing = func() error { return sql.ErrNoRows }
	}

	outcomes := make([]batchweaver.Outcome[V], 0, n)
	for _, rng := range Chunks(n, limits) {
		chunk := items[rng[0]:rng[1]]
		chunkOutcomes, err := p.executeChunk(ctx, chunk, missing, limits)
		if err != nil {
			return batchweaver.BatchResponse[V]{}, err
		}
		outcomes = append(outcomes, chunkOutcomes...)
	}
	return batchweaver.NewBatchResponse(outcomes)
}

// executeChunk runs one chunk and maps rows to outcomes by ordinal.
func (p SQLProvider[K, V]) executeChunk(ctx context.Context, chunk []batchweaver.BatchItem[K], missing func() error, limits ParameterLimits) ([]batchweaver.Outcome[V], error) {
	keys := make([]K, len(chunk))
	for i, it := range chunk {
		keys[i] = it.Key
	}
	args := []any{keys}
	if p.KeyArgs != nil {
		args = p.KeyArgs(keys)
	} else if p.KeyArg != nil {
		args[0] = p.KeyArg(keys)
	}
	wantArgs := len(p.Plan.KeyParams)
	if wantArgs == 0 {
		wantArgs = 1
	}
	if len(args) != wantArgs {
		return nil, fmt.Errorf("%w: plan requires %d key arguments, builder returned %d", ErrProviderMisconfigured, wantArgs, len(args))
	}
	if wantArgs > limits.MaxParameters {
		return nil, fmt.Errorf("%s: batch requires %d parameters, limit is %d", CodeParamLimitExceeded, wantArgs, limits.MaxParameters)
	}
	if p.EstimatePayload != nil && p.EstimatePayload(keys) > limits.MaxPayloadKB*1024 {
		return nil, fmt.Errorf("%s: encoded key payload exceeds %d KiB", CodeParamLimitExceeded, limits.MaxPayloadKB)
	}
	rows, err := p.DB.QueryContext(ctx, p.Plan.Query, args...)
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
		if ord < 1 || int(ord) > len(chunk) {
			return nil, fmt.Errorf("row decode: ordinal %d outside chunk [1,%d]", ord, len(chunk))
		}
		if _, duplicate := values[ord]; duplicate {
			return nil, fmt.Errorf("row decode: duplicate ordinal %d violates scalar cardinality", ord)
		}
		values[ord] = v
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

var (
	_ Queryer = (*sql.DB)(nil)
	_ Queryer = (*sql.Tx)(nil)
	_ Queryer = (*sql.Conn)(nil)
)
