package analysis

import (
	"context"
	"os"
	"runtime"
	"sort"
	"time"

	"golang.org/x/tools/go/packages"
)

// Request describes an analysis to perform.
type Request struct {
	// Patterns are Go package patterns (for example "./...").
	Patterns []string
	// BuildContext selects the build configuration.
	BuildContext BuildContext
	// Reproducible omits volatile fields (timestamps) for byte-stable output.
	Reproducible bool
	// ToolVersion is the BatchWeaver version recorded in the snapshot.
	ToolVersion string
	// Dir is the working directory used to resolve patterns; empty means the
	// process working directory.
	Dir string
}

// Analyze loads the requested packages, discovers operations, builds SSA and a
// conservative call graph, summarizes effects, indexes scalar-operation call
// sites, inventories candidates, and returns a deterministic, immutable
// snapshot. A non-nil error is returned only when package loading cannot be
// performed at all; ordinary package and analysis problems are reported as
// diagnostics in the snapshot.
func Analyze(ctx context.Context, req Request) (*Snapshot, error) {
	patterns := req.Patterns
	if len(patterns) == 0 {
		patterns = []string{"./..."}
	}
	l, loadDiags, err := loadPackages(ctx, patterns, req.BuildContext, req.Dir)
	if err != nil {
		return nil, err
	}
	if err := ctx.Err(); err != nil {
		return nil, err
	}

	pkgs, appPaths, appCount, testVariants, totalLoaded := classifyPackages(l)

	r := newResolver(l.pkgs)
	ops, discDiags, configDigest := discoverOperations(ctx, l, r)

	sr := buildSSA(l)
	appFuncs := appFunctions(sr, appPaths)
	asyncFuncs := computeAsyncFunctions(sr.funcs)
	edges := buildCallGraph(sr, appFuncs)
	sites, byOp := callSiteDiscovery(l, sr, ops, appFuncs, asyncFuncs)
	effects := computeEffects(sr, appFuncs)

	for i := range ops {
		ids := byOp[ops[i].ID]
		sort.Strings(ids)
		ops[i].CallSiteIDs = ids
	}
	candidates := buildCandidates(ops, sites)

	diags := append([]Diag{}, loadDiags...)
	diags = append(diags, discDiags...)
	sortDiags(diags)

	snap := &Snapshot{
		SchemaVersion:       SchemaVersion,
		ToolVersion:         req.ToolVersion,
		GoVersion:           runtime.Version(),
		Workspace:           ".",
		ConfigDigest:        configDigest,
		BuildDigest:         l.bc.digest(),
		PackagesLoaded:      totalLoaded,
		ApplicationPackages: appCount,
		TestVariants:        testVariants,
		SSAFunctions:        sr.funcCount,
		CallGraphEdges:      len(edges),
		Packages:            pkgs,
		Operations:          ops,
		CallSites:           sites,
		Edges:               edges,
		Effects:             effects,
		Candidates:          candidates,
		Diagnostics:         diags,
	}
	if !req.Reproducible {
		snap.Timestamp = time.Now().UTC().Format(time.RFC3339)
	}
	return snap, nil
}

// classifyPackages builds portable package records for the top-level packages
// and returns the application-package path set and counts.
func classifyPackages(l *loaded) (recs []Package, appPaths map[string]bool, appCount, testVariants, totalLoaded int) {
	appPaths = make(map[string]bool)
	visited := make(map[string]bool)
	packages.Visit(l.pkgs, nil, func(p *packages.Package) { visited[p.ID] = true })
	totalLoaded = len(visited)

	for _, p := range l.pkgs {
		class := classify(p)
		if class == ClassApplication || class == ClassCommand {
			appPaths[p.PkgPath] = true
			appCount++
		}
		if class == ClassApplicationTest || isTestVariant(p) {
			appPaths[p.PkgPath] = true
			testVariants++
		}
		files := make([]string, 0, len(p.GoFiles))
		for _, f := range p.GoFiles {
			files = append(files, l.pc.portable(f))
		}
		sort.Strings(files)
		module := ""
		if p.Module != nil {
			module = p.Module.Path
		}
		rec := Package{
			ID:         shortDigest("pkg", p.PkgPath, module, p.ID),
			ImportPath: p.PkgPath,
			Name:       p.Name,
			Module:     module,
			Class:      class,
			Files:      files,
			Generated:  scanGenerated(p),
			HasErrors:  len(p.Errors) > 0,
		}
		for _, e := range p.Errors {
			rec.Errors = append(rec.Errors, e.Msg)
		}
		recs = append(recs, rec)
	}
	sort.Slice(recs, func(i, j int) bool {
		if recs[i].ImportPath != recs[j].ImportPath {
			return recs[i].ImportPath < recs[j].ImportPath
		}
		return recs[i].ID < recs[j].ID
	})
	return recs, appPaths, appCount, testVariants, totalLoaded
}

// classify returns the package classification.
func classify(p *packages.Package) string {
	switch {
	case p.Module != nil && p.Module.Main:
		switch {
		case isTestVariant(p):
			return ClassApplicationTest
		case p.Name == "main":
			return ClassCommand
		default:
			return ClassApplication
		}
	case p.Module == nil:
		return ClassStandardLibrary
	default:
		return ClassDependency
	}
}

// scanGenerated reports whether a package's first source file carries the
// standard generated-code marker. It reads a bounded prefix.
func scanGenerated(p *packages.Package) bool {
	for _, f := range p.GoFiles {
		data, err := os.ReadFile(f)
		if err != nil {
			continue
		}
		if len(data) > 4096 {
			data = data[:4096]
		}
		for _, line := range splitLines(string(data)) {
			if len(line) > 0 && line[0] == '/' && containsGeneratedMarker(line) {
				return true
			}
		}
	}
	return false
}

func splitLines(s string) []string {
	var out []string
	start := 0
	for i := 0; i < len(s); i++ {
		if s[i] == '\n' {
			out = append(out, s[start:i])
			start = i + 1
		}
	}
	if start < len(s) {
		out = append(out, s[start:])
	}
	return out
}

func containsGeneratedMarker(line string) bool {
	const marker = "Code generated"
	const suffix = "DO NOT EDIT."
	return indexOf(line, marker) >= 0 && indexOf(line, suffix) >= 0
}

func indexOf(s, sub string) int {
	for i := 0; i+len(sub) <= len(s); i++ {
		if s[i:i+len(sub)] == sub {
			return i
		}
	}
	return -1
}

// sortDiags orders diagnostics deterministically.
func sortDiags(d []Diag) {
	sort.Slice(d, func(i, j int) bool {
		if d[i].Location != d[j].Location {
			return d[i].Location < d[j].Location
		}
		if d[i].Severity != d[j].Severity {
			return severityRank(d[i].Severity) < severityRank(d[j].Severity)
		}
		if d[i].Code != d[j].Code {
			return d[i].Code < d[j].Code
		}
		return d[i].Fingerprint < d[j].Fingerprint
	})
}

func severityRank(s string) int {
	switch s {
	case "error":
		return 0
	case "warning":
		return 1
	case "info":
		return 2
	default:
		return 3
	}
}
