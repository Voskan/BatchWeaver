package adaptiveruntime

import "testing"

func TestDemoIsDeterministicAndShadow(t *testing.T) {
	a := Demo()
	b := Demo()
	if a != b {
		t.Fatalf("demo is not deterministic: %+v vs %+v", a, b)
	}
	if a.Applied {
		t.Error("shadow-mode demo must never apply a change")
	}
	if a.Operation != "users.get" {
		t.Errorf("operation = %q", a.Operation)
	}
	if a.RecommendWait > a.CurrentWait {
		t.Errorf("balanced objective on a fast backend should not raise wait: %s -> %s", a.CurrentWait, a.RecommendWait)
	}
}
