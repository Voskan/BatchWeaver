// Package service exercises scalar operation call sites for analysis fixtures.
package service

import (
	"context"

	"example.com/fixture/users"
)

// Service depends on a user repository.
type Service struct {
	repo *users.Repository
}

// LoadAll calls the scalar operation inside a range loop (a loop candidate).
func (s *Service) LoadAll(ctx context.Context, ids []users.UserID) ([]users.User, error) {
	var out []users.User
	for _, id := range ids {
		u, err := s.repo.GetUser(ctx, id)
		if err != nil {
			return nil, err
		}
		out = append(out, u)
	}
	return out, nil
}

// LoadAsync calls the scalar operation inside launched goroutines (a fan-out
// candidate).
func (s *Service) LoadAsync(ctx context.Context, ids []users.UserID) {
	for _, id := range ids {
		id := id
		go func() {
			_, _ = s.repo.GetUser(ctx, id)
		}()
	}
}
