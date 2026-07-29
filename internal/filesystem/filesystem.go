// Package filesystem provides a minimal filesystem abstraction used by
// BatchWeaver's project path resolution.
//
// The interface is intentionally tiny: it exposes only the operations that
// project discovery actually needs, so that discovery can be exercised in
// tests against a controlled directory tree without assuming anything about the
// real working directory. It is not a general-purpose virtual filesystem and
// must not grow speculative methods.
package filesystem

import (
	"io/fs"
	"os"
	"path/filepath"
)

// FS is the small set of filesystem operations required by project discovery.
type FS interface {
	// Getwd returns the current working directory.
	Getwd() (string, error)
	// Stat returns file information for the named path.
	Stat(name string) (fs.FileInfo, error)
	// ReadFile reads the entire named file.
	ReadFile(name string) ([]byte, error)
	// Abs returns an absolute representation of path.
	Abs(path string) (string, error)
}

// osFS is the FS implementation backed by the real operating system.
type osFS struct{}

// OS returns an FS backed by the operating system's real filesystem.
func OS() FS { return osFS{} }

func (osFS) Getwd() (string, error)                { return os.Getwd() }
func (osFS) Stat(name string) (fs.FileInfo, error) { return os.Stat(name) }
func (osFS) ReadFile(name string) ([]byte, error)  { return os.ReadFile(name) }
func (osFS) Abs(path string) (string, error)       { return filepath.Abs(path) }
