// Package svc exercises runtime-lowering candidate classes.
package svc

import (
	"context"

	"example.com/rtfixture/repo"
)

// Service depends on a repository.
type Service struct {
	repo *repo.Repo
}

// LoadAll reads each user in a loop (a runtime-lowerable read-only loop).
func (s *Service) LoadAll(ctx context.Context, ids []int) ([]repo.User, error) {
	var out []repo.User
	for _, id := range ids {
		u, err := s.repo.GetUser(ctx, id)
		if err != nil {
			return nil, err
		}
		out = append(out, u)
	}
	return out, nil
}

// LoadPair reads two users as straight-line sibling calls.
func (s *Service) LoadPair(ctx context.Context, a, b int) (repo.User, repo.User, error) {
	ua, err := s.repo.GetUser(ctx, a)
	if err != nil {
		return repo.User{}, repo.User{}, err
	}
	ub, err := s.repo.GetUser(ctx, b)
	if err != nil {
		return repo.User{}, repo.User{}, err
	}
	return ua, ub, nil
}
