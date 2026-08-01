package multioperationwave

import (
	"reflect"
	"testing"
)

func TestDemoWaves(t *testing.T) {
	p := Demo()
	if len(p.Waves) != 3 {
		t.Fatalf("want 3 waves, got %d: %v", len(p.Waves), p.Waves)
	}
	if len(p.Waves[0]) != 2 {
		t.Errorf("wave 0 should co-schedule two independent operations: %v", p.Waves[0])
	}
	want := []string{"load_user", "load_perms", "render"}
	if !reflect.DeepEqual(p.CriticalPath, want) {
		t.Errorf("critical path = %v, want %v", p.CriticalPath, want)
	}
}
