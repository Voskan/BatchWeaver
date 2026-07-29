package operation

import (
	"encoding/json"
	"errors"
	"strings"
	"testing"
)

func TestParseIDValid(t *testing.T) {
	t.Parallel()
	valid := []string{
		"users.get",
		"users.get-by-id",
		"pricing.region.lookup",
		"orders.write.atomic",
		"a.b",
		"a1.b2.c3",
		"user_data.get",
	}
	for _, v := range valid {
		if _, err := ParseID(v); err != nil {
			t.Errorf("ParseID(%q) = %v, want nil", v, err)
		}
	}
}

func TestParseIDInvalid(t *testing.T) {
	t.Parallel()
	invalid := []string{
		"",
		"single",
		"Users.get",
		"users.Get",
		".users.get",
		"users.get.",
		"users..get",
		"users.get-",
		"users.-get",
		"users.ge--t",
		"users.ge__t",
		"users.ge-_t",
		"1users.get",
		"users .get",
		"batchweaver.internal",
		"internal.thing",
		"system.thing",
		strings.Repeat("a", 64) + ".b",
	}
	for _, v := range invalid {
		if _, err := ParseID(v); !errors.Is(err, ErrInvalidID) {
			t.Errorf("ParseID(%q) error = %v, want ErrInvalidID", v, err)
		}
	}
}

func TestIDLengthBoundary(t *testing.T) {
	t.Parallel()
	// A 63-char segment is allowed; 64 is not.
	if _, err := ParseID("a." + strings.Repeat("b", 63)); err != nil {
		t.Errorf("63-char segment rejected: %v", err)
	}
	if _, err := ParseID("a." + strings.Repeat("b", 64)); err == nil {
		t.Errorf("64-char segment accepted")
	}
	// Whole-ID length limit.
	long := "a." + strings.Repeat("b", 62) + "." + strings.Repeat("c", 62)
	_ = long
	tooLong := strings.Repeat("a.", 128) + "b" // exceeds 255 bytes
	if _, err := ParseID(tooLong); err == nil {
		t.Errorf("over-length id accepted")
	}
}

func TestMustParseIDPanics(t *testing.T) {
	t.Parallel()
	defer func() {
		if r := recover(); r == nil {
			t.Errorf("MustParseID did not panic on invalid input")
		}
	}()
	_ = MustParseID("bad")
}

func TestIDSegments(t *testing.T) {
	t.Parallel()
	id := MustParseID("pricing.region.lookup")
	got := id.Segments()
	want := []string{"pricing", "region", "lookup"}
	if len(got) != len(want) {
		t.Fatalf("Segments() = %v, want %v", got, want)
	}
	for i := range want {
		if got[i] != want[i] {
			t.Errorf("segment %d = %q, want %q", i, got[i], want[i])
		}
	}
	if ID("bad").Segments() != nil {
		t.Errorf("Segments() of invalid id should be nil")
	}
}

func TestIDJSONRoundTrip(t *testing.T) {
	t.Parallel()
	type holder struct {
		Op ID `json:"op"`
	}
	in := holder{Op: MustParseID("users.get")}
	data, err := json.Marshal(in)
	if err != nil {
		t.Fatalf("marshal: %v", err)
	}
	var out holder
	if err := json.Unmarshal(data, &out); err != nil {
		t.Fatalf("unmarshal: %v", err)
	}
	if out.Op != in.Op {
		t.Errorf("round trip = %q, want %q", out.Op, in.Op)
	}
	var bad holder
	if err := json.Unmarshal([]byte(`{"op":"Bad.ID"}`), &bad); err == nil {
		t.Errorf("unmarshal invalid id accepted")
	}
}

func FuzzParseID(f *testing.F) {
	for _, s := range []string{"users.get", "", "a.b.c", "Bad.ID", "x..y", "internal.x"} {
		f.Add(s)
	}
	f.Fuzz(func(t *testing.T, s string) {
		id, err := ParseID(s)
		if err != nil {
			return
		}
		// A successfully parsed ID must round-trip and revalidate.
		if id.Validate() != nil {
			t.Errorf("ParseID accepted %q but Validate rejects it", s)
		}
		if id.String() != s {
			t.Errorf("ParseID(%q).String() = %q", s, id.String())
		}
	})
}
