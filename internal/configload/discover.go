// Package configload discovers, reads, and include-expands BatchWeaver
// configuration files into a single merged node tree.
//
// It enforces safety limits (file size, include depth, total files and bytes),
// forbids remote includes, resolves symlinks before cycle detection, and never
// reads home-directory or environment configuration implicitly.
package configload

import (
	"os"
	"path/filepath"

	"github.com/Voskan/BatchWeaver/diagnostics"
	"github.com/Voskan/BatchWeaver/internal/configdecode"
)

// candidateNames are the configuration file names searched during discovery, in
// priority order.
var candidateNames = []string{"batchweaver.yaml", "batchweaver.yml", "batchweaver.json"}

// Discover searches upward from startDir for a configuration file, stopping at
// the filesystem root or at stopAtRoot (when non-empty). It returns the path of
// the single discovered file. If a directory contains more than one candidate it
// reports an ambiguity diagnostic; if none is found it reports a not-found
// diagnostic. The returned bool reports whether a path was found.
func Discover(startDir, stopAtRoot string, diags *diagnostics.Collection) (string, bool) {
	dir, err := filepath.Abs(startDir)
	if err != nil {
		diags.Add(diagAt(configdecode.CodeNotFound, diagnostics.Position{},
			"cannot resolve working directory: "+err.Error()))
		return "", false
	}
	stop := ""
	if stopAtRoot != "" {
		if abs, err := filepath.Abs(stopAtRoot); err == nil {
			stop = abs
		}
	}
	for {
		var present []string
		for _, name := range candidateNames {
			if fileExists(filepath.Join(dir, name)) {
				present = append(present, name)
			}
		}
		switch {
		case len(present) > 1:
			diags.Add(diagAt(configdecode.CodeAmbiguous, diagnostics.Position{File: dir},
				"multiple configuration files found in the same directory: "+join(present)))
			return "", false
		case len(present) == 1:
			return filepath.Join(dir, present[0]), true
		}
		if dir == stop {
			break
		}
		parent := filepath.Dir(dir)
		if parent == dir {
			break
		}
		dir = parent
	}
	diags.Add(diagAt(configdecode.CodeNotFound, diagnostics.Position{},
		"no configuration file found (expected one of "+join(candidateNames)+")"))
	return "", false
}

// fileExists reports whether path exists and is a regular file.
func fileExists(path string) bool {
	info, err := os.Stat(path)
	return err == nil && info.Mode().IsRegular()
}

// join renders a list of names for a diagnostic message.
func join(names []string) string {
	out := ""
	for i, n := range names {
		if i > 0 {
			out += ", "
		}
		out += n
	}
	return out
}

// diagAt builds an error-severity config diagnostic at pos.
func diagAt(code diagnostics.Code, pos diagnostics.Position, msg string) diagnostics.Diagnostic {
	return diagnostics.Diagnostic{
		Code:     code,
		Severity: diagnostics.SeverityError,
		Message:  msg,
		Source:   "config",
		Range:    diagnostics.AtPosition(pos),
	}
}
