// Package buildinfo exposes a stable, deterministic model of the information
// that identifies a BatchWeaver build: its version, source revision, build
// timestamp, and the toolchain and platform it was produced with.
//
// The mutable package-level variables (version, commit, buildDate) are the
// injection points for `-ldflags -X` overrides used by release builds. Local
// and `go run` builds fall back to safe placeholder values so that output is
// never empty and never depends on the surrounding environment.
package buildinfo

import (
	"fmt"
	"runtime"
	"strings"
)

// These variables are overridden at link time by release builds using, for
// example:
//
//	-X github.com/Voskan/BatchWeaver/internal/buildinfo.version=v0.1.0
//	-X github.com/Voskan/BatchWeaver/internal/buildinfo.commit=<sha>
//	-X github.com/Voskan/BatchWeaver/internal/buildinfo.buildDate=<timestamp>
//
// They are intentionally unexported so that the linker-flag surface stays
// small and callers depend on the derived Info value instead.
var (
	version   = defaultVersion
	commit    = defaultUnknown
	buildDate = defaultUnknown
)

const (
	// defaultVersion is the version reported before the first tagged release.
	defaultVersion = "dev"
	// defaultUnknown is the placeholder used when build metadata is absent.
	defaultUnknown = "unknown"
)

// Info is an immutable snapshot of build and platform identification.
//
// Values are captured once via Get; construct additional values only in tests.
type Info struct {
	// Version is the semantic version, or "dev" before the first release.
	Version string
	// Commit is the source revision the binary was built from, if known.
	Commit string
	// BuildDate is the build timestamp, if known.
	BuildDate string
	// Dirty reports whether the source tree had uncommitted changes at build time.
	Dirty bool
	// GoVersion is the Go toolchain version that produced the binary.
	GoVersion string
	// GOOS is the target operating system.
	GOOS string
	// GOARCH is the target architecture.
	GOARCH string
}

// Get returns the Info for the current binary. The version, commit, and build
// date reflect any link-time overrides; the toolchain and platform fields are
// read from the runtime and are therefore always populated.
func Get() Info {
	return Info{
		Version:   version,
		Commit:    commit,
		BuildDate: buildDate,
		Dirty:     false,
		GoVersion: runtime.Version(),
		GOOS:      runtime.GOOS,
		GOARCH:    runtime.GOARCH,
	}
}

// Platform returns the "GOOS/GOARCH" pair, for example "darwin/arm64".
func (i Info) Platform() string {
	return i.GOOS + "/" + i.GOARCH
}

// String returns a deterministic multi-line rendering of the build
// information, suitable for the `version` command. The layout is stable so
// that callers and tests can rely on it.
func (i Info) String() string {
	version := i.Version
	if i.Dirty {
		version += " (dirty)"
	}

	var b strings.Builder
	fmt.Fprintf(&b, "BatchWeaver %s\n", version)
	fmt.Fprintf(&b, "Go: %s\n", i.GoVersion)
	fmt.Fprintf(&b, "Platform: %s\n", i.Platform())
	fmt.Fprintf(&b, "Commit: %s\n", i.Commit)
	fmt.Fprintf(&b, "Build date: %s", i.BuildDate)
	return b.String()
}
