// Package users is an analysis fixture with a scalar and batch method.
package users

import "context"

// UserID identifies a user.
type UserID int

// User is a user record.
type User struct {
	ID   UserID
	Name string
}

// Repository is an in-memory user store.
type Repository struct{}

// GetUser is the scalar operation.
func (r *Repository) GetUser(ctx context.Context, id UserID) (User, error) {
	return User{ID: id, Name: "u"}, nil
}

// GetUsersBatch is the batch operation.
func (r *Repository) GetUsersBatch(ctx context.Context, ids []UserID) ([]User, error) {
	out := make([]User, len(ids))
	for i, id := range ids {
		out[i] = User{ID: id, Name: "u"}
	}
	return out, nil
}
