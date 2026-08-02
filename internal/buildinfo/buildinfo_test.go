package buildinfo

import (
	"runtime"
	"runtime/debug"
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

func TestResolveModuleMetadata(t *testing.T) {
	t.Parallel()

	info := &debug.BuildInfo{
		Main: debug.Module{Path: modulePath, Version: "v0.1.0-beta.2"},
		Settings: []debug.BuildSetting{
			{Key: "vcs.revision", Value: strings.Repeat("a", 40)},
		},
	}
	version, commit := resolveModuleMetadata(defaultVersion, defaultUnknown, info)
	if version != "0.1.0-beta.2" {
		t.Fatalf("version = %q", version)
	}
	if commit != strings.Repeat("a", 40) {
		t.Fatalf("commit = %q", commit)
	}

	explicitVersion, explicitCommit := resolveModuleMetadata("1.2.3", "release", info)
	if explicitVersion != "1.2.3" || explicitCommit != "release" {
		t.Fatalf("explicit metadata was overridden: %q %q", explicitVersion, explicitCommit)
	}

	local := &debug.BuildInfo{
		Main:     debug.Module{Path: modulePath, Version: "(devel)"},
		Settings: info.Settings,
	}
	localVersion, localCommit := resolveModuleMetadata(defaultVersion, defaultUnknown, local)
	if localVersion != defaultVersion || localCommit != defaultUnknown {
		t.Fatalf("local metadata was promoted to a release: %q %q", localVersion, localCommit)
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
		"Channel: stable",
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
