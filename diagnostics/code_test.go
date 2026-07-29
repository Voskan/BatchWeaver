package diagnostics

import (
	"encoding/json"
	"errors"
	"testing"
)

func TestCodeValidate(t *testing.T) {
	t.Parallel()
	valid := []Code{"BWCFG001", "BWCFG021", "BWOP001", "BWDECL001", "BWINT001", "BWCLI099"}
	for _, c := range valid {
		if err := c.Validate(); err != nil {
			t.Errorf("Validate(%q) = %v, want nil", c, err)
		}
	}
	invalid := []Code{"", "BW001", "bwcfg001", "BWCFG21", "BWCFG0021", "BW_CFG001", "CFG001", "BWCFG001 "}
	for _, c := range invalid {
		if err := c.Validate(); !errors.Is(err, ErrInvalidCode) {
			t.Errorf("Validate(%q) error = %v, want ErrInvalidCode", c, err)
		}
	}
}

func TestCodeCategory(t *testing.T) {
	t.Parallel()
	if got := Code("BWCFG021").Category(); got != "CFG" {
		t.Errorf("Category() = %q, want CFG", got)
	}
	if got := Code("BWDECL001").Category(); got != "DECL" {
		t.Errorf("Category() = %q, want DECL", got)
	}
	if got := Code("bad").Category(); got != "" {
		t.Errorf("Category() of invalid = %q, want empty", got)
	}
}

func TestCodeJSONRoundTrip(t *testing.T) {
	t.Parallel()
	c := Code("BWCFG021")
	data, err := json.Marshal(c)
	if err != nil {
		t.Fatalf("marshal: %v", err)
	}
	if string(data) != `"BWCFG021"` {
		t.Errorf("marshal = %s, want \"BWCFG021\"", data)
	}
	var back Code
	if err := json.Unmarshal(data, &back); err != nil || back != c {
		t.Errorf("round trip = %q, %v", back, err)
	}
	var bad Code
	if err := json.Unmarshal([]byte(`"nope"`), &bad); !errors.Is(err, ErrInvalidCode) {
		t.Errorf("unmarshal invalid code error = %v, want ErrInvalidCode", err)
	}
}
