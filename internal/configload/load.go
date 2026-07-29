package configload

import (
	"context"
	"fmt"
	"os"
	"path/filepath"
	"strings"

	"github.com/Voskan/BatchWeaver/diagnostics"
	"github.com/Voskan/BatchWeaver/internal/configdecode"
	"github.com/Voskan/BatchWeaver/internal/configmerge"
)

// Default limits, applied when the corresponding Options field is zero.
const (
	defaultMaxFileSize   = 1 << 20 // 1 MiB per file
	defaultMaxDepth      = 16      // include nesting depth
	defaultMaxTotalFiles = 128     // total files loaded
	defaultMaxTotalBytes = 8 << 20 // 8 MiB total decoded bytes
)

// includeKey is the top-level field listing local include files.
const includeKey = "include"

// Options controls discovery and loading.
type Options struct {
	// Path is an explicit configuration file path; when empty, discovery is used.
	Path string
	// WorkingDirectory is the base for discovery and relative display paths.
	WorkingDirectory string
	// Discover enables upward search when Path is empty.
	Discover bool
	// AllowAbsoluteInclude permits absolute include paths.
	AllowAbsoluteInclude bool
	// RepositoryRoot, when set, stops discovery at that directory.
	RepositoryRoot string
	// MaximumFileSize bounds a single file's size in bytes.
	MaximumFileSize int64
	// MaximumIncludeDepth bounds include nesting depth.
	MaximumIncludeDepth int
	// MaximumTotalFiles bounds the number of files loaded.
	MaximumTotalFiles int
	// MaximumTotalBytes bounds total decoded bytes.
	MaximumTotalBytes int64
}

// withDefaults returns a copy of opts with zero limit fields replaced by
// defaults.
func (o Options) withDefaults() Options {
	if o.MaximumFileSize == 0 {
		o.MaximumFileSize = defaultMaxFileSize
	}
	if o.MaximumIncludeDepth == 0 {
		o.MaximumIncludeDepth = defaultMaxDepth
	}
	if o.MaximumTotalFiles == 0 {
		o.MaximumTotalFiles = defaultMaxTotalFiles
	}
	if o.MaximumTotalBytes == 0 {
		o.MaximumTotalBytes = defaultMaxTotalBytes
	}
	if o.WorkingDirectory == "" {
		if wd, err := os.Getwd(); err == nil {
			o.WorkingDirectory = wd
		}
	}
	return o
}

// Result is the outcome of loading and merging configuration.
type Result struct {
	// Node is the merged configuration node, or nil when loading failed.
	Node *configdecode.Node
	// Files lists the user-facing paths loaded, in load order.
	Files []string
	// Diagnostics holds loading diagnostics.
	Diagnostics diagnostics.Collection
	// Found reports whether a configuration file was located.
	Found bool
}

// Load discovers (when needed), reads, and include-expands configuration into a
// single merged node. Filesystem failures for the primary file are returned as
// an error; user-facing problems (syntax, includes, limits) are diagnostics.
func Load(ctx context.Context, opts Options) (Result, error) {
	opts = opts.withDefaults()
	var res Result

	path := opts.Path
	if path == "" {
		if !opts.Discover {
			res.Diagnostics.Add(diagAt(configdecode.CodeNotFound, diagnostics.Position{},
				"no configuration path provided and discovery is disabled"))
			return res, nil
		}
		found, ok := Discover(opts.WorkingDirectory, opts.RepositoryRoot, &res.Diagnostics)
		if !ok {
			return res, nil
		}
		path = found
	}
	abs, err := filepath.Abs(path)
	if err != nil {
		return res, fmt.Errorf("resolve configuration path %q: %w", path, err)
	}
	// An explicitly named primary file that does not exist is treated as
	// not-found rather than as an invalid configuration.
	if !fileExists(abs) {
		res.Diagnostics.Add(diagAt(configdecode.CodeNotFound, diagnostics.Position{File: path},
			fmt.Sprintf("configuration file not found: %q", path)))
		return res, nil
	}
	res.Found = true

	l := &loader{ctx: ctx, opts: opts, visited: make(map[string]bool)}
	node := l.expand(abs, nil)
	res.Node = node
	res.Files = l.files
	res.Diagnostics = l.diags
	if node == nil {
		res.Node = nil
	}
	return res, l.fatal
}

// loader carries state across recursive include expansion.
type loader struct {
	ctx        context.Context
	opts       Options
	diags      diagnostics.Collection
	stack      []string // resolved absolute paths currently being expanded
	visited    map[string]bool
	files      []string
	totalFiles int
	totalBytes int64
	fatal      error
}

// expand reads and include-expands the file at absolute path abs. includePos is
// the position of the include directive that referenced it, for diagnostics.
func (l *loader) expand(abs string, includePos *diagnostics.Position) *configdecode.Node {
	if l.ctx != nil {
		if err := l.ctx.Err(); err != nil {
			l.fatal = err
			return nil
		}
	}
	resolved := resolveSymlinks(abs)

	// Cycle detection against the current expansion stack.
	for _, s := range l.stack {
		if s == resolved {
			l.diags.Add(cycleDiag(includePos, append(append([]string{}, l.stackDisplay()...), l.display(abs))))
			return nil
		}
	}
	if l.visited[resolved] {
		// Already merged elsewhere; load once.
		return nil
	}
	if len(l.stack) >= l.opts.MaximumIncludeDepth {
		l.diags.Add(diagAt(configdecode.CodeLimitExceeded, posOrEmpty(includePos),
			fmt.Sprintf("include depth exceeds limit of %d", l.opts.MaximumIncludeDepth)))
		return nil
	}
	if l.totalFiles >= l.opts.MaximumTotalFiles {
		l.diags.Add(diagAt(configdecode.CodeLimitExceeded, posOrEmpty(includePos),
			fmt.Sprintf("total files exceed limit of %d", l.opts.MaximumTotalFiles)))
		return nil
	}

	display := l.display(abs)
	src, ok := l.readFile(abs, display, includePos)
	if !ok {
		return nil
	}

	node, diags := parse(display, src)
	l.diags.AddCollection(diags)
	l.files = append(l.files, display)
	l.totalFiles++
	l.visited[resolved] = true
	if node == nil {
		return nil
	}

	includes := extractIncludes(node)
	body := stripInclude(node)

	l.stack = append(l.stack, resolved)
	defer func() { l.stack = l.stack[:len(l.stack)-1] }()

	var merged *configdecode.Node
	for _, inc := range includes {
		incAbs, ok := l.resolveInclude(abs, inc)
		if !ok {
			continue
		}
		incPos := inc.pos
		expanded := l.expand(incAbs, &incPos)
		merged = configmerge.Merge(merged, expanded, &l.diags)
	}
	merged = configmerge.Merge(merged, body, &l.diags)
	return merged
}

// includeRef is a single include entry with its source position.
type includeRef struct {
	path string
	pos  diagnostics.Position
}

// resolveInclude validates and resolves an include path relative to the
// including file. It rejects remote URLs and (by default) absolute paths.
func (l *loader) resolveInclude(fromAbs string, inc includeRef) (string, bool) {
	if strings.Contains(inc.path, "://") {
		l.diags.Add(diagAt(configdecode.CodeRemoteInclude, inc.pos,
			fmt.Sprintf("remote includes are not allowed: %q", inc.path)))
		return "", false
	}
	if filepath.IsAbs(inc.path) {
		if !l.opts.AllowAbsoluteInclude {
			l.diags.Add(diagAt(configdecode.CodeAbsoluteInclude, inc.pos,
				fmt.Sprintf("absolute includes are not allowed: %q", inc.path)))
			return "", false
		}
		return inc.path, true
	}
	return filepath.Join(filepath.Dir(fromAbs), inc.path), true
}

// readFile reads abs, enforcing per-file and total byte limits.
func (l *loader) readFile(abs, display string, includePos *diagnostics.Position) ([]byte, bool) {
	info, err := os.Stat(abs)
	if err != nil {
		l.diags.Add(diagAt(configdecode.CodeIncludeError, posOrEmpty(includePos),
			fmt.Sprintf("cannot read %q: %v", display, err)))
		return nil, false
	}
	if info.Size() > l.opts.MaximumFileSize {
		l.diags.Add(diagAt(configdecode.CodeLimitExceeded, diagnostics.Position{File: display},
			fmt.Sprintf("file %q exceeds size limit of %d bytes", display, l.opts.MaximumFileSize)))
		return nil, false
	}
	if l.totalBytes+info.Size() > l.opts.MaximumTotalBytes {
		l.diags.Add(diagAt(configdecode.CodeLimitExceeded, diagnostics.Position{File: display},
			fmt.Sprintf("total decoded bytes exceed limit of %d", l.opts.MaximumTotalBytes)))
		return nil, false
	}
	data, err := os.ReadFile(abs)
	if err != nil {
		l.diags.Add(diagAt(configdecode.CodeIncludeError, diagnostics.Position{File: display},
			fmt.Sprintf("cannot read %q: %v", display, err)))
		return nil, false
	}
	l.totalBytes += int64(len(data))
	return data, true
}

// display returns a user-facing path relative to the working directory when
// possible, otherwise the absolute path.
func (l *loader) display(abs string) string {
	if l.opts.WorkingDirectory != "" {
		if rel, err := filepath.Rel(l.opts.WorkingDirectory, abs); err == nil && !strings.HasPrefix(rel, "..") {
			return rel
		}
	}
	return abs
}

// stackDisplay returns display paths for the current stack.
func (l *loader) stackDisplay() []string {
	out := make([]string, len(l.stack))
	for i, s := range l.stack {
		out[i] = l.display(s)
	}
	return out
}

// parse decodes bytes by file extension: .json uses the JSON decoder, everything
// else uses YAML.
func parse(display string, src []byte) (*configdecode.Node, diagnostics.Collection) {
	if strings.EqualFold(filepath.Ext(display), ".json") {
		return configdecode.ParseJSON(display, src)
	}
	return configdecode.ParseYAML(display, src)
}

// extractIncludes returns the include entries of a mapping node.
func extractIncludes(node *configdecode.Node) []includeRef {
	v, _, ok := node.Get(includeKey)
	if !ok || !v.IsSequence() {
		return nil
	}
	var out []includeRef
	for _, e := range v.Elems {
		if s, ok := configdecode.AsString(e); ok {
			out = append(out, includeRef{path: s, pos: e.Pos})
		}
	}
	return out
}

// stripInclude returns a copy of node without its include entry.
func stripInclude(node *configdecode.Node) *configdecode.Node {
	if !node.IsMapping() {
		return node
	}
	out := &configdecode.Node{Kind: configdecode.KindMapping, Pos: node.Pos}
	for _, e := range node.Entries {
		if e.Key == includeKey {
			continue
		}
		out.Entries = append(out.Entries, e)
	}
	return out
}

// resolveSymlinks resolves symlinks for cycle detection, falling back to the
// cleaned path when resolution fails (for example, a not-yet-existing file).
func resolveSymlinks(abs string) string {
	if r, err := filepath.EvalSymlinks(abs); err == nil {
		return r
	}
	return filepath.Clean(abs)
}

// posOrEmpty dereferences an optional position.
func posOrEmpty(p *diagnostics.Position) diagnostics.Position {
	if p == nil {
		return diagnostics.Position{}
	}
	return *p
}

// cycleDiag builds an include-cycle diagnostic describing the chain.
func cycleDiag(includePos *diagnostics.Position, chain []string) diagnostics.Diagnostic {
	d := diagAt(configdecode.CodeIncludeCycle, posOrEmpty(includePos),
		"configuration include cycle detected")
	d.Details = strings.Join(chain, " -> ")
	return d
}
