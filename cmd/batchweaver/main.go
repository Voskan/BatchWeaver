// Command batchweaver is the BatchWeaver command-line entry point.
//
// It is intentionally thin: it constructs the CLI application, delegates to it,
// and maps the returned exit code to the process exit status. All behavior
// lives in internal/cli so that it can be tested without spawning a process.
package main

import (
	"context"
	"os"

	"github.com/Voskan/BatchWeaver/internal/cli"
)

func main() {
	app := cli.New(os.Stdout, os.Stderr)
	code := app.Run(context.Background(), os.Args[1:])
	os.Exit(int(code))
}
