// Package repo is a proof-engine fixture with scalar and batch methods across
// read and write operations.
package repo

import "context"

// User is a user record.
type User struct {
	ID   int
	Name string
}

// Repo is an in-memory store.
type Repo struct{}

// GetUser is a read-only scalar operation.
func (r *Repo) GetUser(ctx context.Context, id int) (User, error) {
	return User{ID: id}, nil
}

// GetUsersBatch is the batch form of GetUser.
func (r *Repo) GetUsersBatch(ctx context.Context, ids []int) ([]User, error) {
	out := make([]User, len(ids))
	for i, id := range ids {
		out[i] = User{ID: id}
	}
	return out, nil
}

// GetPrice is a read-only scalar operation keyed by currency.
func (r *Repo) GetPrice(ctx context.Context, currency string) (int, error) {
	return len(currency), nil
}

// GetPricesBatch is the batch form of GetPrice.
func (r *Repo) GetPricesBatch(ctx context.Context, currencies []string) ([]int, error) {
	out := make([]int, len(currencies))
	for i, c := range currencies {
		out[i] = len(c)
	}
	return out, nil
}

// AppendEvent is a non-idempotent write scalar operation.
func (r *Repo) AppendEvent(ctx context.Context, event string) (int, error) {
	return len(event), nil
}

// AppendEventsBatch is the batch form of AppendEvent.
func (r *Repo) AppendEventsBatch(ctx context.Context, events []string) ([]int, error) {
	out := make([]int, len(events))
	for i, e := range events {
		out[i] = len(e)
	}
	return out, nil
}

// GetNext is a read-only scalar operation whose result feeds the next key.
func (r *Repo) GetNext(ctx context.Context, id int) (int, error) {
	return id + 1, nil
}

// GetNextBatch is the batch form of GetNext.
func (r *Repo) GetNextBatch(ctx context.Context, ids []int) ([]int, error) {
	out := make([]int, len(ids))
	for i, id := range ids {
		out[i] = id + 1
	}
	return out, nil
}
