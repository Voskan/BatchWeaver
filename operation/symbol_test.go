package operation

import (
	"errors"
	"testing"
)

func TestParseSymbolFunction(t *testing.T) {
	t.Parallel()
	s, err := ParseSymbol("github.com/example/project/users.GetUser")
	if err != nil {
		t.Fatalf("ParseSymbol: %v", err)
	}
	if s.ImportPath() != "github.com/example/project/users" {
		t.Errorf("ImportPath = %q", s.ImportPath())
	}
	if s.Name() != "GetUser" || s.IsMethod() || s.PointerReceiver() {
		t.Errorf("unexpected function fields: %+v", s)
	}
}

func TestParseSymbolPointerMethod(t *testing.T) {
	t.Parallel()
	s, err := ParseSymbol("github.com/example/project/users.(*Repository).GetUser")
	if err != nil {
		t.Fatalf("ParseSymbol: %v", err)
	}
	if !s.IsMethod() || !s.PointerReceiver() {
		t.Errorf("expected pointer method: %+v", s)
	}
	if s.Receiver() != "Repository" || s.Name() != "GetUser" {
		t.Errorf("unexpected method fields: %+v", s)
	}
	if s.ImportPath() != "github.com/example/project/users" {
		t.Errorf("ImportPath = %q", s.ImportPath())
	}
}

func TestParseSymbolValueMethod(t *testing.T) {
	t.Parallel()
	s, err := ParseSymbol("github.com/example/project/users.(Repository).GetUser")
	if err != nil {
		t.Fatalf("ParseSymbol: %v", err)
	}
	if !s.IsMethod() || s.PointerReceiver() {
		t.Errorf("expected value method: %+v", s)
	}
	if s.Receiver() != "Repository" {
		t.Errorf("Receiver = %q", s.Receiver())
	}
}

func TestParseSymbolStdlib(t *testing.T) {
	t.Parallel()
	s, err := ParseSymbol("fmt.Println")
	if err != nil {
		t.Fatalf("ParseSymbol: %v", err)
	}
	if s.ImportPath() != "fmt" || s.Name() != "Println" {
		t.Errorf("unexpected fields: %+v", s)
	}
}

func TestParseSymbolInvalid(t *testing.T) {
	t.Parallel()
	invalid := []string{
		"",
		"nodot",
		"./relative.Func",
		"../up.Func",
		"/absolute/path.Func",
		`C:\windows.Func`,
		"has space.Func",
		"pkg.(Broken.Method",
		"pkg.*Broken).Method",
		"pkg.(*).Method",
		"pkg.(*Repo).",
		"pkg.(*Repo).1bad",
		"pkg.1bad",
	}
	for _, v := range invalid {
		if _, err := ParseSymbol(v); !errors.Is(err, ErrInvalidSymbol) {
			t.Errorf("ParseSymbol(%q) error = %v, want ErrInvalidSymbol", v, err)
		}
	}
}

func TestSymbolStringRoundTrip(t *testing.T) {
	t.Parallel()
	inputs := []string{
		"github.com/example/project/users.GetUser",
		"github.com/example/project/users.(*Repository).GetUser",
		"github.com/example/project/users.(Repository).GetUser",
		"fmt.Println",
	}
	for _, in := range inputs {
		s, err := ParseSymbol(in)
		if err != nil {
			t.Fatalf("ParseSymbol(%q): %v", in, err)
		}
		if s.String() != in {
			t.Errorf("String() = %q, want %q", s.String(), in)
		}
		reparsed, err := ParseSymbol(s.String())
		if err != nil || reparsed != s {
			t.Errorf("re-parse mismatch for %q", in)
		}
	}
}

func TestSymbolTextRoundTrip(t *testing.T) {
	t.Parallel()
	s := MustParseSymbol("github.com/example/project/users.(*Repository).GetUser")
	data, err := s.MarshalText()
	if err != nil {
		t.Fatalf("MarshalText: %v", err)
	}
	var back Symbol
	if err := back.UnmarshalText(data); err != nil || back != s {
		t.Errorf("round trip = %+v, %v", back, err)
	}
}

func TestZeroSymbolInvalid(t *testing.T) {
	t.Parallel()
	var s Symbol
	if !s.IsZero() {
		t.Errorf("zero symbol IsZero = false")
	}
	if err := s.Validate(); err == nil {
		t.Errorf("zero symbol Validate = nil, want error")
	}
}

func FuzzParseSymbol(f *testing.F) {
	for _, s := range []string{
		"fmt.Println",
		"github.com/a/b.(*R).M",
		"github.com/a/b.(R).M",
		"",
		"bad",
		"pkg.(*).X",
	} {
		f.Add(s)
	}
	f.Fuzz(func(t *testing.T, s string) {
		sym, err := ParseSymbol(s)
		if err != nil {
			return
		}
		if sym.String() != s {
			t.Errorf("ParseSymbol(%q).String() = %q", s, sym.String())
		}
		if sym.Validate() != nil {
			t.Errorf("ParseSymbol accepted %q but Validate rejects it", s)
		}
	})
}
