package buildinfo

import (
	"runtime"
	"strings"
	"testing"
)

func TestGetDefaults(t *testing.T) {
	t.Parallel()

	got := Get()

	if got.Version != defaultVersion {
		t.Errorf("Version = %q, want %q", got.Version, defaultVersion)
	}
	if got.Commit != defaultUnknown {
		t.Errorf("Commit = %q, want %q", got.Commit, defaultUnknown)
	}
	if got.BuildDate != defaultUnknown {
		t.Errorf("BuildDate = %q, want %q", got.BuildDate, defaultUnknown)
	}
	if got.GoVersion != runtime.Version() {
		t.Errorf("GoVersion = %q, want %q", got.GoVersion, runtime.Version())
	}
	if got.GOOS != runtime.GOOS {
		t.Errorf("GOOS = %q, want %q", got.GOOS, runtime.GOOS)
	}
	if got.GOARCH != runtime.GOARCH {
		t.Errorf("GOARCH = %q, want %q", got.GOARCH, runtime.GOARCH)
	}
}

func TestPlatform(t *testing.T) {
	t.Parallel()

	info := Info{GOOS: "linux", GOARCH: "amd64"}
	if got, want := info.Platform(), "linux/amd64"; got != want {
		t.Errorf("Platform() = %q, want %q", got, want)
	}
}

func TestStringPopulated(t *testing.T) {
	t.Parallel()

	info := Info{
		Version:   "v1.2.3",
		Commit:    "abc1234",
		BuildDate: "2026-01-02T03:04:05Z",
		GoVersion: "go1.26.5",
		GOOS:      "linux",
		GOARCH:    "amd64",
	}

	got := info.String()
	want := strings.Join([]string{
		"BatchWeaver v1.2.3",
		"Go: go1.26.5",
		"Platform: linux/amd64",
		"Commit: abc1234",
		"Build date: 2026-01-02T03:04:05Z",
	}, "\n")

	if got != want {
		t.Errorf("String() =\n%q\nwant\n%q", got, want)
	}
}

func TestStringDirty(t *testing.T) {
	t.Parallel()

	info := Info{Version: "dev", Dirty: true, GoVersion: "go1.26.5", GOOS: "darwin", GOARCH: "arm64"}
	got := info.String()
	if !strings.Contains(got, "BatchWeaver dev (dirty)") {
		t.Errorf("String() = %q, want it to mark the build dirty", got)
	}
}
