// Package svc exercises proof-engine candidate classes.
package svc

import (
	"context"

	"example.com/prooffixture/repo"
)

// Service depends on a repository.
type Service struct {
	repo *repo.Repo
}

// currencyCursor is package-global mutable state observed between iterations.
var currencyCursor int

// nextCurrency mutates a package global and is therefore an observable effect.
func nextCurrency() string {
	currencyCursor++
	return "usd"
}

// SafeLoop is a read-only loop over an invariant receiver and context with a
// structural key. It is expected to be proven eligible for static loop prefetch.
func (s *Service) SafeLoop(ctx context.Context, ids []int) ([]repo.User, error) {
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

// WriteLoop calls a non-idempotent write in a loop. It is expected to be proven
// ineligible because writes are not batched by the core engine.
func (s *Service) WriteLoop(ctx context.Context, events []string) error {
	for _, e := range events {
		if _, err := s.repo.AppendEvent(ctx, e); err != nil {
			return err
		}
	}
	return nil
}

// ChainLoop computes each key from the previous operation result. Static prefetch
// is expected to be ineligible because the key is loop-carried through the
// operation result.
func (s *Service) ChainLoop(ctx context.Context, start int) (int, error) {
	id := start
	for i := 0; i < 10; i++ {
		next, err := s.repo.GetNext(ctx, id)
		if err != nil {
			return 0, err
		}
		id = next
	}
	return id, nil
}

// PriceLoop computes the key by calling a function that mutates a package
// global. Static movement is expected to be unknown due to the observable
// barrier and call-derived key.
func (s *Service) PriceLoop(ctx context.Context, n int) error {
	for i := 0; i < n; i++ {
		if _, err := s.repo.GetPrice(ctx, nextCurrency()); err != nil {
			return err
		}
	}
	return nil
}

// Getter abstracts the scalar read behind an interface, producing an ambiguous
// target that the engine must not optimistically resolve.
type Getter interface {
	GetUser(context.Context, int) (repo.User, error)
}

// IfaceLoop dispatches through an interface. The target is unresolved, so even
// runtime coalescing eligibility is unknown.
func (s *Service) IfaceLoop(ctx context.Context, g Getter, ids []int) error {
	for _, id := range ids {
		if _, err := g.GetUser(ctx, id); err != nil {
			return err
		}
	}
	return nil
}
