package recursivebatching

import (
	"reflect"
	"testing"
)

func TestDemoBreadthFirst(t *testing.T) {
	r := Demo()
	// A complete binary tree loaded to depth 4: frontiers 1, 2, 4, 8, 16.
	want := []int{1, 2, 4, 8, 16}
	if !reflect.DeepEqual(r.FrontierSizes, want) {
		t.Errorf("frontier sizes = %v, want %v", r.FrontierSizes, want)
	}
	if r.BackendCalls != len(want) {
		t.Errorf("backend calls = %d, want one per frontier (%d)", r.BackendCalls, len(want))
	}
	if r.Nodes != 31 {
		t.Errorf("nodes = %d, want 31", r.Nodes)
	}
}
