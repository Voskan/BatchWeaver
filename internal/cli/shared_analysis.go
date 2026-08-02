package cli

import (
	"context"
	"errors"

	"github.com/Voskan/BatchWeaver/internal/analysis"
	"github.com/Voskan/BatchWeaver/internal/analysiscache"
	"github.com/Voskan/BatchWeaver/internal/daemon"
)

// sharedAnalyze uses the local workspace daemon when compatible and otherwise
// performs the same request in-process. The fallback preserves CLI availability;
// no result from a failed or mismatched daemon is reused.
func sharedAnalyze(ctx context.Context, request analysis.Request) (*analysis.Snapshot, analysiscache.Result, error) {
	root := request.Dir
	if root == "" {
		root = cwd()
	}
	result, err := daemon.Analyze(ctx, root, daemon.AnalysisParams{
		Patterns: request.Patterns, BuildContext: request.BuildContext,
		Reproducible: request.Reproducible, ToolVersion: request.ToolVersion,
		Overlay: request.Overlay,
	})
	if err == nil {
		return result.Snapshot, result.Cache, nil
	}
	source := "fallback"
	if errors.Is(err, daemon.ErrNotRunning) || errors.Is(err, daemon.ErrStale) {
		source = "local"
	}
	snapshot, localErr := analysis.Analyze(ctx, request)
	return snapshot, analysiscache.Result{Source: source}, localErr
}
