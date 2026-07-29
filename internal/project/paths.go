// Package project resolves BatchWeaver's project locations, most importantly
// the repository root, without assuming any particular current working
// directory.
//
// Root discovery walks upward from a starting directory looking for well-known
// markers (a go.mod file or a .git entry). This keeps commands robust when they
// are run from a subdirectory of a project.
package project

import (
	"errors"
	"fmt"
	"path/filepath"

	"github.com/Voskan/BatchWeaver/internal/filesystem"
)

// ErrRootNotFound is returned when no repository root marker is found while
// walking up from the starting directory.
var ErrRootNotFound = errors.New("project root not found")

// rootMarkers are the entries whose presence in a directory identifies it as a
// project root. The go.mod file is authoritative for a Go module; .git handles
// the case of a checkout whose module lives elsewhere.
var rootMarkers = []string{"go.mod", ".git"}

// Locator resolves project paths against a filesystem. Construct it with New so
// that tests can supply an in-memory or temporary-directory filesystem.
type Locator struct {
	fs filesystem.FS
}

// New returns a Locator backed by fsys. If fsys is nil, the real operating
// system filesystem is used.
func New(fsys filesystem.FS) *Locator {
	if fsys == nil {
		fsys = filesystem.OS()
	}
	return &Locator{fs: fsys}
}

// FindRoot returns the nearest ancestor of start (inclusive) that contains a
// project root marker. The start directory is resolved to an absolute path
// first, so relative inputs are accepted. It returns ErrRootNotFound if no
// marker is present up to the filesystem root.
func (l *Locator) FindRoot(start string) (string, error) {
	dir, err := l.fs.Abs(start)
	if err != nil {
		return "", fmt.Errorf("resolve start directory %q: %w", start, err)
	}

	for {
		if l.hasMarker(dir) {
			return dir, nil
		}
		parent := filepath.Dir(dir)
		if parent == dir {
			// Reached the filesystem root without finding a marker.
			return "", fmt.Errorf("%w: searched up to %q", ErrRootNotFound, dir)
		}
		dir = parent
	}
}

// RootFromWorkingDir resolves the project root starting from the current
// working directory.
func (l *Locator) RootFromWorkingDir() (string, error) {
	wd, err := l.fs.Getwd()
	if err != nil {
		return "", fmt.Errorf("determine working directory: %w", err)
	}
	return l.FindRoot(wd)
}

// hasMarker reports whether dir directly contains any root marker.
func (l *Locator) hasMarker(dir string) bool {
	for _, marker := range rootMarkers {
		if _, err := l.fs.Stat(filepath.Join(dir, marker)); err == nil {
			return true
		}
	}
	return false
}
