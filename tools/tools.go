//go:build tools

// Package tools pins the versions of BatchWeaver's development tools via blank
// imports. It is never built into any BatchWeaver binary; the `tools` build tag
// excludes it from normal builds. It exists so that tool versions are recorded
// in this module's go.mod and stay reproducible across contributors and CI.
package tools

import (
	_ "github.com/golangci/golangci-lint/v2/cmd/golangci-lint"
	_ "golang.org/x/vuln/cmd/govulncheck"
)
