package diagnostics

import (
	"errors"
	"testing"
)

func TestPositionValidateAndString(t *testing.T) {
	t.Parallel()
	tests := []struct {
		name    string
		pos     Position
		wantErr bool
		wantStr string
	}{
		{"zero", Position{}, false, "-"},
		{"file only", Position{File: "a.yaml"}, false, "a.yaml"},
		{"file line", Position{File: "a.yaml", Line: 12}, false, "a.yaml:12"},
		{"full", Position{File: "a.yaml", Line: 12, Column: 5}, false, "a.yaml:12:5"},
		{"negative", Position{Line: -1}, true, ""},
		{"column without line", Position{File: "a.yaml", Column: 3}, true, ""},
	}
	for _, tt := range tests {
		err := tt.pos.Validate()
		if tt.wantErr {
			if !errors.Is(err, ErrInvalidPosition) {
				t.Errorf("%s: Validate() = %v, want ErrInvalidPosition", tt.name, err)
			}
			continue
		}
		if err != nil {
			t.Errorf("%s: Validate() = %v, want nil", tt.name, err)
		}
		if got := tt.pos.String(); got != tt.wantStr {
			t.Errorf("%s: String() = %q, want %q", tt.name, got, tt.wantStr)
		}
	}
}

func TestRangeValidate(t *testing.T) {
	t.Parallel()
	ok := Range{Start: Position{File: "a", Line: 1, Column: 1}, End: Position{File: "a", Line: 2, Column: 1}}
	if err := ok.Validate(); err != nil {
		t.Errorf("valid range error: %v", err)
	}
	bad := Range{Start: Position{File: "a", Line: 5}, End: Position{File: "a", Line: 2}}
	if err := bad.Validate(); !errors.Is(err, ErrInvalidPosition) {
		t.Errorf("end-before-start error = %v, want ErrInvalidPosition", err)
	}
}

func TestDiagnosticValidate(t *testing.T) {
	t.Parallel()
	valid := Diagnostic{Code: "BWCFG001", Severity: SeverityError, Message: "boom", Source: "config"}
	if err := valid.Validate(); err != nil {
		t.Errorf("valid diagnostic error: %v", err)
	}

	cases := map[string]Diagnostic{
		"bad code":           {Code: "BAD", Severity: SeverityError, Message: "m"},
		"unset severity":     {Code: "BWCFG001", Message: "m"},
		"empty message":      {Code: "BWCFG001", Severity: SeverityInfo, Message: "  "},
		"trailing space":     {Code: "BWCFG001", Severity: SeverityInfo, Message: "m "},
		"newline in message": {Code: "BWCFG001", Severity: SeverityInfo, Message: "a\nb"},
		"control char":       {Code: "BWCFG001", Severity: SeverityInfo, Message: "a\x07b"},
	}
	for name, d := range cases {
		if err := d.Validate(); err == nil {
			t.Errorf("%s: Validate() = nil, want error", name)
		}
	}
}

func TestDiagnosticString(t *testing.T) {
	t.Parallel()
	d := Diagnostic{
		Code:     "BWCFG021",
		Severity: SeverityError,
		Message:  `unknown field "max_batch_items"`,
		Range:    Range{Start: Position{File: "batchweaver.yaml", Line: 27, Column: 5}},
	}
	want := `batchweaver.yaml:27:5: error BWCFG021: unknown field "max_batch_items"`
	if got := d.String(); got != want {
		t.Errorf("String() = %q, want %q", got, want)
	}
}

func TestRelatedInformationValidate(t *testing.T) {
	t.Parallel()
	if err := (RelatedInformation{Message: "prior definition", Range: Range{Start: Position{File: "a", Line: 1}}}).Validate(); err != nil {
		t.Errorf("valid related error: %v", err)
	}
	if err := (RelatedInformation{Message: " "}).Validate(); !errors.Is(err, ErrInvalidDiagnostic) {
		t.Errorf("empty related message error = %v, want ErrInvalidDiagnostic", err)
	}
}
