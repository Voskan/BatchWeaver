package config

import (
	"errors"
	"testing"
	"time"
)

func TestParseDuration(t *testing.T) {
	t.Parallel()
	valid := map[string]time.Duration{
		"0s":    0,
		"200us": 200 * time.Microsecond,
		"500µs": 500 * time.Microsecond,
		"5ms":   5 * time.Millisecond,
		"2s":    2 * time.Second,
		"1m30s": 90 * time.Second,
	}
	for in, want := range valid {
		d, err := ParseDuration(in)
		if err != nil {
			t.Errorf("ParseDuration(%q) = %v", in, err)
			continue
		}
		if d.Std() != want {
			t.Errorf("ParseDuration(%q) = %v, want %v", in, d.Std(), want)
		}
	}
}

func TestParseDurationRejectsBareNumbers(t *testing.T) {
	t.Parallel()
	for _, in := range []string{"", "5", "100", "-3", "abc"} {
		if _, err := ParseDuration(in); !errors.Is(err, ErrInvalidDuration) {
			t.Errorf("ParseDuration(%q) error = %v, want ErrInvalidDuration", in, err)
		}
	}
}

func TestDurationCanonicalString(t *testing.T) {
	t.Parallel()
	cases := map[string]string{
		"0s":    "0s",
		"500us": "500µs",
		"1m30s": "1m30s",
	}
	for in, want := range cases {
		d, err := ParseDuration(in)
		if err != nil {
			t.Fatalf("ParseDuration(%q): %v", in, err)
		}
		if got := d.String(); got != want {
			t.Errorf("Duration(%q).String() = %q, want %q", in, got, want)
		}
	}
}

func TestDurationTextRoundTrip(t *testing.T) {
	t.Parallel()
	d := Duration(90 * time.Second)
	data, _ := d.MarshalText()
	var back Duration
	if err := back.UnmarshalText(data); err != nil || back != d {
		t.Errorf("round trip = %v, %v", back, err)
	}
}

func FuzzParseDuration(f *testing.F) {
	for _, s := range []string{"5ms", "0s", "500us", "", "abc", "-1s", "100"} {
		f.Add(s)
	}
	f.Fuzz(func(t *testing.T, s string) {
		d, err := ParseDuration(s)
		if err != nil {
			return
		}
		// Canonical string must re-parse to the same value.
		d2, err := ParseDuration(d.String())
		if err != nil || d2 != d {
			t.Errorf("ParseDuration(%q)=%v not stable under re-parse (%v, %v)", s, d, d2, err)
		}
	})
}
