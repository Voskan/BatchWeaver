package config

import (
	"context"

	"github.com/Voskan/BatchWeaver/diagnostics"
	"github.com/Voskan/BatchWeaver/internal/configload"
	"github.com/Voskan/BatchWeaver/operation"
)

// LoadOptions controls configuration discovery and loading.
type LoadOptions struct {
	// Path is an explicit configuration file path. When empty and Discover is
	// true, the loader searches upward from WorkingDirectory.
	Path string
	// WorkingDirectory is the base directory for discovery and for computing
	// user-facing relative paths. When empty, the process working directory is
	// used.
	WorkingDirectory string
	// Discover enables upward search for a configuration file when Path is empty.
	Discover bool
	// AllowAbsoluteInclude permits absolute include paths (disabled by default).
	AllowAbsoluteInclude bool
	// RepositoryRoot, when set, stops discovery at that directory.
	RepositoryRoot string
	// MaximumFileSize bounds a single file's size in bytes (0 uses the default).
	MaximumFileSize int64
	// MaximumIncludeDepth bounds include nesting depth (0 uses the default).
	MaximumIncludeDepth int
}

// LoadResult is the outcome of loading configuration. Invalid user configuration
// is reported through Diagnostics with a partial Config, rather than through the
// error return.
type LoadResult struct {
	// Config is the normalized effective configuration.
	Config Config
	// Catalog is the normalized operation catalog (also available via Config).
	Catalog operation.Catalog
	// Diagnostics holds all findings from loading, normalization, and validation.
	Diagnostics diagnostics.Collection
	// Files lists the configuration files loaded, in load order, as user-facing
	// paths.
	Files []string
	// Digest is the semantic configuration digest, or empty when the
	// configuration has errors.
	Digest string
	// Found reports whether a configuration file was located.
	Found bool
}

// HasErrors reports whether the load produced any error-severity diagnostics.
func (r LoadResult) HasErrors() bool { return r.Diagnostics.HasErrors() }

// Load discovers, reads, include-expands, merges, normalizes, and validates
// configuration. It respects context cancellation. Filesystem failures for the
// primary file and internal invariant failures are returned as an error; all
// user-facing configuration problems are returned as diagnostics in the result.
func Load(ctx context.Context, options LoadOptions) (LoadResult, error) {
	loaded, err := configload.Load(ctx, configload.Options{
		Path:                 options.Path,
		WorkingDirectory:     options.WorkingDirectory,
		Discover:             options.Discover,
		AllowAbsoluteInclude: options.AllowAbsoluteInclude,
		RepositoryRoot:       options.RepositoryRoot,
		MaximumFileSize:      options.MaximumFileSize,
		MaximumIncludeDepth:  options.MaximumIncludeDepth,
	})
	if err != nil {
		return LoadResult{Found: loaded.Found, Diagnostics: loaded.Diagnostics, Files: loaded.Files}, err
	}

	res := LoadResult{Found: loaded.Found, Files: loaded.Files}
	res.Diagnostics.AddCollection(loaded.Diagnostics)

	if loaded.Node == nil {
		return res, nil
	}

	cfg, catalog, normDiags := normalize(loaded.Node)
	res.Diagnostics.AddCollection(normDiags)
	res.Diagnostics.AddCollection(validateSemantic(cfg))
	res.Config = cfg
	res.Catalog = catalog

	if !res.Diagnostics.HasErrors() {
		res.Digest = Digest(cfg)
	}
	return res, nil
}
