// Package repo is a runtime-lowering fixture store.
package repo

import "context"

// User is a user record.
type User struct {
	ID   int
	Name string
}

// Repo is an in-memory store.
type Repo struct{}

// GetUser is the read-only scalar operation.
func (r *Repo) GetUser(ctx context.Context, id int) (User, error) {
	return User{ID: id}, nil
}

// GetUsersBatch is the batch form.
func (r *Repo) GetUsersBatch(ctx context.Context, ids []int) ([]User, error) {
	out := make([]User, len(ids))
	for i, id := range ids {
		out[i] = User{ID: id}
	}
	return out, nil
}
