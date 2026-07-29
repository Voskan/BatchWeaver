package diagnostics

import (
	"encoding/json"
	"errors"
	"testing"
)

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
	for _, s := range []Severity{SeverityInfo, SeverityWarning, SeverityError} {
		if !s.Valid() {
			t.Errorf("Severity(%d).Valid() = false, want true", s)
		}
	}
	for _, s := range []Severity{SeverityUnknown, Severity(42)} {
		if s.Valid() {
			t.Errorf("Severity(%d).Valid() = true, want false", s)
		}
	}
}

func TestParseSeverity(t *testing.T) {
	t.Parallel()
	for in, want := range map[string]Severity{
		"unknown": SeverityUnknown,
		"info":    SeverityInfo,
		"warning": SeverityWarning,
		"error":   SeverityError,
	} {
		got, err := ParseSeverity(in)
		if err != nil || got != want {
			t.Errorf("ParseSeverity(%q) = %v, %v; want %v, nil", in, got, err, want)
		}
	}
	for _, bad := range []string{"", "Error", "ERROR", "warn", " info"} {
		if _, err := ParseSeverity(bad); !errors.Is(err, ErrInvalidSeverity) {
			t.Errorf("ParseSeverity(%q) error = %v, want ErrInvalidSeverity", bad, err)
		}
	}
}

func TestSeverityJSONRoundTrip(t *testing.T) {
	t.Parallel()
	for _, s := range []Severity{SeverityInfo, SeverityWarning, SeverityError, SeverityUnknown} {
		data, err := json.Marshal(s)
		if err != nil {
			t.Fatalf("marshal %v: %v", s, err)
		}
		var back Severity
		if err := json.Unmarshal(data, &back); err != nil {
			t.Fatalf("unmarshal %s: %v", data, err)
		}
		if back != s {
			t.Errorf("round trip %v -> %s -> %v", s, data, back)
		}
	}
	var bad Severity
	if err := json.Unmarshal([]byte(`"nope"`), &bad); !errors.Is(err, ErrInvalidSeverity) {
		t.Errorf("unmarshal invalid severity error = %v, want ErrInvalidSeverity", err)
	}
}

func TestSeverityMarshalInvalid(t *testing.T) {
	t.Parallel()
	if _, err := Severity(99).MarshalText(); !errors.Is(err, ErrInvalidSeverity) {
		t.Errorf("MarshalText(99) error = %v, want ErrInvalidSeverity", err)
	}
}
