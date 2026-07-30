// Package staticprefetch demonstrates the BatchWeaver static-loop-prefetch
// transformation. Scalar is the original N+1 loop; Prefetched is exactly the
// shape BatchWeaver's `transform` command generates for a certified read-only
// slice loop over an ordered, global-error batch provider.
//
// The two functions are proven behaviorally equivalent by example_test.go. The
// only intended difference is structural: Scalar issues one backend call per id,
// while Prefetched issues a single batch call.
package staticprefetch

import (
	"context"

	"github.com/Voskan/BatchWeaver/examples/static-prefetch/repo"
)

// Order references a user by ID; it is the loop element whose field supplies the
// operation key.
type Order struct {
	ID     int
	UserID int
}

// Scalar loads each order's user with one scalar call per order (the original
// N+1 loop).
func Scalar(ctx context.Context, s *repo.Store, orders []Order) ([]repo.User, error) {
	var out []repo.User
	for _, order := range orders {
		u, err := s.GetUser(ctx, order.UserID)
		if err != nil {
			return nil, err
		}
		out = append(out, u)
	}
	return out, nil
}

// Prefetched is the transformed equivalent produced by static loop prefetch: it
// collects keys in source order, performs one batch call, validates the global
// result, and replays the original loop body in source order.
func Prefetched(ctx context.Context, s *repo.Store, orders []Order) ([]repo.User, error) {
	// BatchWeaver: static prefetch for operation users.get.
	var out []repo.User
	bwKeys := make([]int, 0, len(orders))
	for _, order := range orders {
		bwKeys = append(bwKeys, order.UserID)
	}
	bwValues, bwErr := s.GetUsersBatch(ctx, bwKeys)
	if bwErr != nil {
		return nil, bwErr
	}
	bwIndex := 0
	for _, order := range orders {
		_ = order
		u, err := bwValues[bwIndex], error(nil)
		bwIndex++
		if err != nil {
			return nil, err
		}
		out = append(out, u)
	}
	return out, nil
}
