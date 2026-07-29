// This is a separate module that pins BatchWeaver's development tool versions.
// It is intentionally kept apart from the main module so that tool dependencies
// do not enter the main module's go.mod/go.sum. See tools/README.md.
module github.com/Voskan/BatchWeaver/tools

go 1.26

toolchain go1.26.5

require (
	github.com/golangci/golangci-lint/v2 v2.12.2
	golang.org/x/vuln v1.6.0
)
