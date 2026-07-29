package config

import (
	"errors"
	"testing"
)

func TestParseByteSize(t *testing.T) {
	t.Parallel()
	valid := map[string]int64{
		"0B":    0,
		"64KiB": 64 * 1024,
		"4MiB":  4 * 1024 * 1024,
		"1GiB":  1024 * 1024 * 1024,
		"1KB":   1000,
		"2MB":   2 * 1000 * 1000,
	}
	for in, want := range valid {
		b, err := ParseByteSize(in)
		if err != nil {
			t.Errorf("ParseByteSize(%q) = %v", in, err)
			continue
		}
		if b.Bytes() != want {
			t.Errorf("ParseByteSize(%q) = %d, want %d", in, b.Bytes(), want)
		}
	}
}

func TestParseByteSizeRejects(t *testing.T) {
	t.Parallel()
	for _, in := range []string{"", "1024", "10", "-4MiB", "4XiB", "MiB", "abc"} {
		if _, err := ParseByteSize(in); !errors.Is(err, ErrInvalidByteSize) {
			t.Errorf("ParseByteSize(%q) error = %v, want ErrInvalidByteSize", in, err)
		}
	}
}

func TestByteSizeCanonicalString(t *testing.T) {
	t.Parallel()
	cases := map[string]string{
		"0B":    "0B",
		"64KiB": "64KiB",
		"4MiB":  "4MiB",
		"1024B": "1KiB",
		"1KB":   "1000B",
	}
	for in, want := range cases {
		b, err := ParseByteSize(in)
		if err != nil {
			t.Fatalf("ParseByteSize(%q): %v", in, err)
		}
		if got := b.String(); got != want {
			t.Errorf("ByteSize(%q).String() = %q, want %q", in, got, want)
		}
	}
}

func TestByteSizeOverflow(t *testing.T) {
	t.Parallel()
	if _, err := ParseByteSize("9999999999999999999GiB"); err == nil {
		t.Errorf("overflow not detected")
	}
}

func FuzzParseByteSize(f *testing.F) {
	for _, s := range []string{"64KiB", "0B", "4MiB", "", "1024", "-1B", "abc"} {
		f.Add(s)
	}
	f.Fuzz(func(t *testing.T, s string) {
		b, err := ParseByteSize(s)
		if err != nil {
			return
		}
		if b.Bytes() < 0 {
			t.Errorf("ParseByteSize(%q) produced negative %d", s, b.Bytes())
		}
		b2, err := ParseByteSize(b.String())
		if err != nil || b2 != b {
			t.Errorf("ParseByteSize(%q)=%d not stable under re-parse (%v,%v)", s, b.Bytes(), b2, err)
		}
	})
}
