// Package main is an editor fixture for the transformation-preview flow. Invoke
// "BatchWeaver: Preview Transformation" (or the code lens) to open a read-only
// preview of the candidate; no source is modified until you explicitly apply.
package main

import "fmt"

// Repo is a stand-in scalar backend.
type Repo struct{ users map[int]string }

// User returns one user (the scalar operation).
func (r *Repo) User(id int) (string, error) {
	if v, ok := r.users[id]; ok {
		return v, nil
	}
	return "", fmt.Errorf("no user %d", id)
}

func main() {
	r := &Repo{users: map[int]string{10: "ada", 20: "grace"}}
	for _, id := range []int{10, 20} {
		u, err := r.User(id)
		if err == nil {
			fmt.Println(u)
		}
	}
}
