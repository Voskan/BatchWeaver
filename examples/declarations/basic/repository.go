// Package basic is a compile-tested example of declaring a BatchWeaver
// operation. It shows a scalar method paired with a native batch method and a
// typed declaration that connects them to an operation spec.
//
// Declaration is implemented today. Automatic interception of scalar calls is
// not implemented yet; later compiler prompts will discover eligible call sites
// and transform them. This example only demonstrates the typed contracts.
package basic

import (
	"context"
	"fmt"

	batchweaver "github.com/Voskan/BatchWeaver"
)

// UserID identifies a user.
type UserID int

// User is a stored user record.
type User struct {
	ID   UserID
	Name string
}

// ErrUserNotFound is returned by the scalar method when a user is absent.
var ErrUserNotFound = fmt.Errorf("user not found")

// Repository is an in-memory example data source.
type Repository struct {
	users map[UserID]User
}

// NewRepository returns a Repository seeded with a few users.
func NewRepository() *Repository {
	return &Repository{users: map[UserID]User{
		1: {ID: 1, Name: "Ada"},
		2: {ID: 2, Name: "Grace"},
	}}
}

// GetUser is the scalar operation: it loads a single user by ID.
func (r *Repository) GetUser(_ context.Context, id UserID) (User, error) {
	u, ok := r.users[id]
	if !ok {
		return User{}, ErrUserNotFound
	}
	return u, nil
}

// GetUsersBatch is the native batch operation: it loads many users in one call
// and maps each outcome back to its request ID. Absent users become not-found
// outcomes, matching the operation's result contract.
func (r *Repository) GetUsersBatch(
	_ context.Context,
	req batchweaver.BatchRequest[UserID],
) (batchweaver.BatchResponse[User], error) {
	outcomes := make([]batchweaver.Outcome[User], 0, req.Len())
	for _, item := range req.Items() {
		if u, ok := r.users[item.Key]; ok {
			outcomes = append(outcomes, batchweaver.Success(item.ID, u))
		} else {
			outcomes = append(outcomes, batchweaver.NotFound[User](item.ID))
		}
	}
	return batchweaver.NewBatchResponse(outcomes)
}
