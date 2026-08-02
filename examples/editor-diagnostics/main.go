// Package main is an editor fixture: an N+1 loop that BatchWeaver flags as a
// batching opportunity. Open this folder in an editor with the BatchWeaver
// language server running to see the live diagnostic and code lens. No source is
// modified by opening it.
package main

import "fmt"

// Store is a stand-in scalar backend.
type Store struct{ data map[int]string }

// Get returns one record (the scalar operation).
func (s *Store) Get(id int) (string, error) {
	if v, ok := s.data[id]; ok {
		return v, nil
	}
	return "", fmt.Errorf("not found: %d", id)
}

func main() {
	s := &Store{data: map[int]string{1: "a", 2: "b", 3: "c"}}
	ids := []int{1, 2, 3}
	// N+1: one scalar call per id. BatchWeaver surfaces this as a candidate.
	for _, id := range ids {
		v, err := s.Get(id)
		if err != nil {
			continue
		}
		fmt.Println(v)
	}
}
