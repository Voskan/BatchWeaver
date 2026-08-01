package fairnessoverload

import "testing"

func TestDemoFairnessAndOverload(t *testing.T) {
	r := Demo()
	if r.OverloadState != "critical" {
		t.Errorf("overload state = %q, want critical", r.OverloadState)
	}
	if r.LowPriority != "shed-low-priority" {
		t.Errorf("low-priority action = %q, want shed-low-priority", r.LowPriority)
	}
	if r.CriticalAction != "accept" {
		t.Errorf("critical action = %q, want accept", r.CriticalAction)
	}
	if len(r.Shares) != 3 {
		t.Errorf("want 3 class shares, got %d", len(r.Shares))
	}
}
