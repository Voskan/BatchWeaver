// Package repo is an in-memory user store for the static-prefetch example. It
// provides a scalar read and an ordered, global-error batch read, and counts
// calls so tests can demonstrate structural call elimination.
package repo

import (
	"context"
	"errors"
)

// ErrNotFound is returned for a missing user by both the scalar and batch reads.
var ErrNotFound = errors.New("user not found")

// User is a user record.
type User struct {
	ID   int
	Name string
}

// Store is an in-memory user store.
type Store struct {
	data        map[int]string
	ScalarCalls int
	BatchCalls  int
}

// New returns a store seeded with the given id→name entries.
func New(entries map[int]string) *Store {
	data := make(map[int]string, len(entries))
	for k, v := range entries {
		data[k] = v
	}
	return &Store{data: data}
}

// GetUser is the scalar read. It returns ErrNotFound for a missing id.
func (s *Store) GetUser(_ context.Context, id int) (User, error) {
	s.ScalarCalls++
	name, ok := s.data[id]
	if !ok {
		return User{}, ErrNotFound
	}
	return User{ID: id, Name: name}, nil
}

// GetUsersBatch is the ordered, global-error batch read. It returns one User per
// input id in input order, or (nil, ErrNotFound) if any id is missing, matching
// the first-error behavior of the scalar read under the certified contract.
func (s *Store) GetUsersBatch(_ context.Context, ids []int) ([]User, error) {
	s.BatchCalls++
	out := make([]User, 0, len(ids))
	for _, id := range ids {
		name, ok := s.data[id]
		if !ok {
			return nil, ErrNotFound
		}
		out = append(out, User{ID: id, Name: name})
	}
	return out, nil
}
