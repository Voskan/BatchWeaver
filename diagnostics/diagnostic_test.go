package diagnostics

import (
	"errors"
	"testing"
)

func TestPositionString(t *testing.T) {
	t.Parallel()

	tests := []struct {
		name string
		pos  Position
		want string
	}{
		{"zero", Position{}, "-"},
		{"file only", Position{File: "main.go"}, "main.go"},
		{"file and line", Position{File: "main.go", Line: 12}, "main.go:12"},
		{"full", Position{File: "main.go", Line: 12, Column: 5}, "main.go:12:5"},
		{"line without column", Position{File: "a.go", Line: 3, Column: 0}, "a.go:3"},
		{"column ignored without line", Position{File: "a.go", Column: 5}, "a.go"},
	}

	for _, tt := range tests {
		if got := tt.pos.String(); got != tt.want {
			t.Errorf("%s: Position.String() = %q, want %q", tt.name, got, tt.want)
		}
	}
}

func TestDiagnosticValidate(t *testing.T) {
	t.Parallel()

	valid := Diagnostic{Severity: SeverityError, Message: "something went wrong"}
	if err := valid.Validate(); err != nil {
		t.Errorf("Validate() on valid diagnostic returned error: %v", err)
	}

	cases := []struct {
		name string
		diag Diagnostic
	}{
		{"unset severity", Diagnostic{Message: "msg"}},
		{"invalid severity", Diagnostic{Severity: Severity(99), Message: "msg"}},
		{"empty message", Diagnostic{Severity: SeverityInfo, Message: "   "}},
	}
	for _, tt := range cases {
		err := tt.diag.Validate()
		if err == nil {
			t.Errorf("%s: Validate() = nil, want error", tt.name)
			continue
		}
		if !errors.Is(err, ErrInvalidDiagnostic) {
			t.Errorf("%s: Validate() error = %v, want it to wrap ErrInvalidDiagnostic", tt.name, err)
		}
	}
}

func TestDiagnosticString(t *testing.T) {
	t.Parallel()

	tests := []struct {
		name string
		diag Diagnostic
		want string
	}{
		{
			name: "full",
			diag: Diagnostic{
				Code:     "BW0001",
				Severity: SeverityError,
				Message:  "unbatchable call",
				Position: Position{File: "svc.go", Line: 10, Column: 2},
			},
			want: "svc.go:10:2: error: [BW0001] unbatchable call",
		},
		{
			name: "no position no code",
			diag: Diagnostic{Severity: SeverityWarning, Message: "heads up"},
			want: "warning: heads up",
		},
		{
			name: "code without position",
			diag: Diagnostic{Code: "BW0002", Severity: SeverityInfo, Message: "note"},
			want: "info: [BW0002] note",
		},
	}

	for _, tt := range tests {
		if got := tt.diag.String(); got != tt.want {
			t.Errorf("%s: Diagnostic.String() = %q, want %q", tt.name, got, tt.want)
		}
	}
}
