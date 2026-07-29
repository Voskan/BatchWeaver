package diagnostics

import "testing"

func TestSeverityString(t *testing.T) {
	t.Parallel()

	tests := []struct {
		sev  Severity
		want string
	}{
		{SeverityUnknown, "unknown"},
		{SeverityInfo, "info"},
		{SeverityWarning, "warning"},
		{SeverityError, "error"},
		{Severity(200), "unknown"},
	}

	for _, tt := range tests {
		if got := tt.sev.String(); got != tt.want {
			t.Errorf("Severity(%d).String() = %q, want %q", tt.sev, got, tt.want)
		}
	}
}

func TestSeverityValid(t *testing.T) {
	t.Parallel()

	valid := []Severity{SeverityInfo, SeverityWarning, SeverityError}
	for _, s := range valid {
		if !s.Valid() {
			t.Errorf("Severity(%d).Valid() = false, want true", s)
		}
	}

	invalid := []Severity{SeverityUnknown, Severity(42)}
	for _, s := range invalid {
		if s.Valid() {
			t.Errorf("Severity(%d).Valid() = true, want false", s)
		}
	}
}
