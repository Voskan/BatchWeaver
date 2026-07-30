package transform

import (
	"bytes"
	"context"
	"fmt"
	"go/format"
	"sort"
	"strings"

	"github.com/Voskan/BatchWeaver/config"
	"github.com/Voskan/BatchWeaver/internal/analysis"
	"github.com/Voskan/BatchWeaver/internal/proof"
	"github.com/Voskan/BatchWeaver/operation"
	"golang.org/x/tools/go/packages"
)

// Request describes a transformation-planning run.
type Request struct {
	Patterns    []string
	Dir         string
	Strategies  []StrategyID
	BuildConfig BuildConfig
	Filter      Filter
	ToolVersion string
	Toolchain   string
}

// Filter selects a subset of candidates without altering proof semantics.
type Filter struct {
	Candidate string
	Operation string
	Package   string
	File      string
	Max       int
}

// symbols carries the resolved scalar/batch symbols for an operation.
type symbols struct{ scalar, batch string }

// BuildPlan builds a deterministic transformation plan for the requested
// packages. It consumes proof certificates and transforms only candidates that
// are currently proven eligible for a requested strategy. It never modifies
// source files.
func BuildPlan(ctx context.Context, req Request) (*Plan, error) {
	patterns := req.Patterns
	if len(patterns) == 0 {
		patterns = []string{"./..."}
	}
	strategies := req.Strategies
	if len(strategies) == 0 {
		strategies = []StrategyID{StrategyStaticLoopPrefetch}
	}

	report, syms, contractDigest, err := runProof(ctx, req, patterns)
	if err != nil {
		return nil, err
	}
	l, err := loadForRewrite(ctx, req.Dir, patterns, req.BuildConfig)
	if err != nil {
		return nil, err
	}

	plan := &Plan{
		SchemaVersion:   SchemaVersion,
		Workspace:       ".",
		Toolchain:       req.Toolchain,
		BuildConfig:     req.BuildConfig,
		AnalysisDigest:  report.AnalysisDigest,
		ProofSchema:     report.SchemaVersion,
		ContractDigest:  contractDigest,
		StrategyVersion: StrategyVersion,
		Validation:      ValidationSummary{Parse: ValidationNotRun, TypeCheck: ValidationNotRun, Preconditions: ValidationPassed, Structural: ValidationNotRun},
	}

	wantStatic := false
	for _, s := range strategies {
		if s == StrategyStaticLoopPrefetch {
			wantStatic = true
		}
	}

	fileTransformed := map[string][]byte{} // rel -> transformed bytes
	fileOriginal := map[string][]byte{}    // rel -> original bytes
	fileTaken := map[string]bool{}         // rel already transformed (no composition yet)

	count := 0
	for _, cp := range report.CandidateProofs {
		if req.Filter.Max > 0 && count >= req.Filter.Max {
			break
		}
		if !matchesFilter(req.Filter, cp) {
			continue
		}
		cert := newCertificate(cp)
		if !wantStatic {
			plan.Skipped = append(plan.Skipped, skip(cp, SkipStrategyNotRequested, ""))
			continue
		}
		if ok, reason := cert.eligibleFor(StrategyStaticLoopPrefetch); !ok {
			plan.Skipped = append(plan.Skipped, skip(cp, reason, string(cp.Decision)))
			continue
		}
		sym, ok := syms[cp.Operation]
		if !ok || sym.scalar == "" || sym.batch == "" {
			plan.Skipped = append(plan.Skipped, skip(cp, SkipBindingUnavailable, "operation has no resolved scalar/batch symbols"))
			continue
		}
		site, reason := l.locateLoop(cert.Location, sym.scalar, sym.batch)
		if reason != "" {
			plan.Skipped = append(plan.Skipped, skip(cp, reason, ""))
			continue
		}
		if fileTaken[site.fi.rel] {
			plan.Skipped = append(plan.Skipped, skip(cp, SkipOverlappingRegion, "another transformation already targets this file"))
			continue
		}

		transformed, replacement, startOff, endOff := l.generate(site)
		formatted, ferr := format.Source(transformed)
		if ferr != nil {
			plan.Diagnostics = append(plan.Diagnostics, diag("BW3401", "error",
				"transformed file does not format", site.fi.rel, cp.ID, ferr.Error()))
			continue
		}
		fileTaken[site.fi.rel] = true
		fileTransformed[site.fi.rel] = formatted
		fileOriginal[site.fi.rel] = site.fi.src

		tr := buildTransformation(cp, cert, site, l, startOff, endOff, replacement)
		plan.Transformations = append(plan.Transformations, tr)
		count++
	}

	// Build file plans deterministically.
	var relPaths []string
	for rel := range fileTransformed {
		relPaths = append(relPaths, rel)
	}
	sort.Strings(relPaths)
	for _, rel := range relPaths {
		orig := fileOriginal[rel]
		trans := fileTransformed[rel]
		ins, rem := lineDelta(orig, trans)
		plan.Files = append(plan.Files, FilePlan{
			Path:              rel,
			OriginalDigest:    hashBytes(orig),
			TransformedDigest: hashBytes(trans),
			InsertedLines:     ins,
			RemovedLines:      rem,
			transformed:       trans,
			original:          orig,
		})
	}

	// Validate parse + type-check through an overlay.
	if len(plan.Files) > 0 {
		plan.Validation.Parse = ValidationPassed // format.Source already parsed
		plan.Validation.Structural = structuralOK(plan)
		if err := typeCheckOverlay(ctx, l, plan); err != nil {
			plan.Validation.TypeCheck = ValidationFailed
			plan.Validation.Detail = err.Error()
			plan.Diagnostics = append(plan.Diagnostics, diag("BW3402", "error",
				"transformed package does not type-check", "", "", err.Error()))
		} else {
			plan.Validation.TypeCheck = ValidationPassed
		}
	} else {
		plan.Validation.Parse = ValidationSkipped
		plan.Validation.TypeCheck = ValidationSkipped
		plan.Validation.Structural = ValidationSkipped
	}

	finalizePlanDigest(plan)
	return plan, nil
}

// runProof runs analysis and proof and resolves operation symbols.
func runProof(ctx context.Context, req Request, patterns []string) (*proof.Report, map[string]symbols, string, error) {
	snap, err := analysis.Analyze(ctx, analysis.Request{
		Patterns: patterns, Reproducible: true, ToolVersion: req.ToolVersion, Dir: req.Dir,
		BuildContext: analysis.BuildContext{
			GOOS: req.BuildConfig.GOOS, GOARCH: req.BuildConfig.GOARCH,
			Tags: req.BuildConfig.Tags, Tests: req.BuildConfig.Tests,
		},
	})
	if err != nil {
		return nil, nil, "", err
	}
	specs := map[string]operation.Spec{}
	syms := map[string]symbols{}
	contractDigest := ""
	dir := req.Dir
	if dir == "" {
		dir = "."
	}
	if res, cErr := config.Load(ctx, config.LoadOptions{WorkingDirectory: dir, Discover: true, RepositoryRoot: dir}); cErr == nil && res.Found && !res.HasErrors() {
		for _, sp := range res.Catalog.List() {
			id := sp.ID().String()
			specs[id] = sp
			syms[id] = symbols{scalar: sp.ScalarSymbol().String(), batch: sp.BatchSymbol().String()}
		}
		contractDigest = res.Digest
	}
	report, err := proof.Prove(ctx, proof.Input{
		Snapshot: snap, Specs: specs, ContractDigest: contractDigest,
		Reproducible: true, ToolVersion: req.ToolVersion,
	})
	if err != nil {
		return nil, nil, "", err
	}
	return report, syms, contractDigest, nil
}

// matchesFilter reports whether a candidate passes the selection filters.
func matchesFilter(f Filter, cp proof.CandidateProof) bool {
	if f.Candidate != "" && cp.ID != f.Candidate {
		return false
	}
	if f.Operation != "" && cp.Operation != f.Operation {
		return false
	}
	if f.File != "" && !strings.HasPrefix(cp.Location, f.File) {
		return false
	}
	if f.Package != "" && !strings.Contains(cp.Location, f.Package) {
		return false
	}
	return true
}

// buildTransformation constructs the IR record for one rewrite.
func buildTransformation(cp proof.CandidateProof, cert Certificate, site *loopSite, l *loaded, startOff, endOff int, replacement string) Transformation {
	start := l.fset.Position(site.rng.Pos())
	end := l.fset.Position(site.rng.End())
	anchor := SourceAnchor{
		File: site.fi.rel, Package: site.fi.pkg.PkgPath,
		Function:  funcDeclName(site.fn),
		StartLine: start.Line, StartCol: start.Column,
		EndLine: end.Line, EndCol: end.Column,
		StructuralHash: hashParts("loop", site.fi.rel, cp.Operation, site.batchName)[:shortLen],
		Resolution:     AnchorExact,
	}
	edit := Edit{
		ID: shortID("bwedit", site.fi.rel, cp.ID), File: site.fi.rel, Kind: EditReplaceRange,
		StartOffset: startOff, EndOffset: endOff,
		OriginalDigest: hashBytes([]byte(srcOfRange(site, l, startOff, endOff))),
		Replacement:    replacement, Order: startOff,
	}
	names := []string{site.names.keys, site.names.values, site.names.err, site.names.index}
	tr := Transformation{
		CandidateID: cp.ID, CertificateID: cp.ProofID, Strategy: StrategyStaticLoopPrefetch,
		Operation: cp.Operation, Source: anchor,
		Phases:           []Phase{PhaseCollectKeys, PhaseInvokeBatch, PhaseValidateGlobal, PhaseMapResults, PhaseReplayScalarOrder, PhaseExecuteOriginal, PhaseFinalize},
		GeneratedSymbols: names,
		Edits:            []string{edit.ID},
		Assumptions:      cert.Assumptions,
		NonGuarantees:    cert.NonGuarantees,
	}
	tr.ID = shortID("bwtransform", cp.ID, cp.ProofID, string(tr.Strategy))
	tr.Digest = hashParts(tr.ID, tr.CandidateID, tr.CertificateID, string(tr.Strategy),
		anchor.StructuralHash, strings.Join(names, ","), edit.OriginalDigest, hashBytes([]byte(replacement)))
	return tr
}

func srcOfRange(site *loopSite, l *loaded, start, end int) string {
	return string(site.fi.src[start:end])
}

// skip builds a skipped-candidate record.
func skip(cp proof.CandidateProof, reason, detail string) SkippedCandidate {
	return SkippedCandidate{CandidateID: cp.ID, Operation: cp.Operation, Reason: reason, Detail: detail}
}

// diag builds a transformation diagnostic.
func diag(code, severity, message, location, candidate, remediation string) Diagnostic {
	return Diagnostic{
		Code: code, Severity: severity, Message: message, Location: location,
		Candidate: candidate, Remediation: remediation,
		Fingerprint: shortID("bwtfp", code, location, candidate),
	}
}

// lineDelta counts inserted and removed lines between two byte slices using a
// simple line-set delta sufficient for the summary counts.
func lineDelta(orig, trans []byte) (inserted, removed int) {
	o := bytes.Count(orig, []byte{'\n'})
	t := bytes.Count(trans, []byte{'\n'})
	if t > o {
		inserted = t - o
	} else {
		removed = o - t
	}
	return inserted, removed
}

// structuralOK performs light structural verification of the static-prefetch
// output: each transformation must reference the four generated symbols.
func structuralOK(plan *Plan) ValidationState {
	for _, tr := range plan.Transformations {
		if len(tr.GeneratedSymbols) != 4 {
			return ValidationFailed
		}
	}
	return ValidationPassed
}

// typeCheckOverlay type-checks the affected packages against an in-memory
// overlay of the transformed files.
func typeCheckOverlay(ctx context.Context, l *loaded, plan *Plan) error {
	overlay := map[string][]byte{}
	pkgPaths := map[string]bool{}
	for _, fp := range plan.Files {
		fi := l.byFile[fp.Path]
		if fi == nil {
			return fmt.Errorf("overlay: unknown file %s", fp.Path)
		}
		overlay[fi.abs] = fp.transformed
		pkgPaths[fi.pkg.PkgPath] = true
	}
	cfg := &packages.Config{
		Mode:    packages.NeedName | packages.NeedFiles | packages.NeedSyntax | packages.NeedTypes | packages.NeedTypesInfo | packages.NeedImports | packages.NeedDeps,
		Context: ctx, Dir: l.root, Overlay: overlay,
	}
	var patterns []string
	for p := range pkgPaths {
		patterns = append(patterns, p)
	}
	sort.Strings(patterns)
	pkgs, err := packages.Load(cfg, patterns...)
	if err != nil {
		return err
	}
	var errs []string
	packages.Visit(pkgs, nil, func(p *packages.Package) {
		for _, e := range p.Errors {
			errs = append(errs, e.Error())
		}
	})
	if len(errs) > 0 {
		sort.Strings(errs)
		return fmt.Errorf("%s", strings.Join(errs, "; "))
	}
	return nil
}

// finalizePlanDigest computes the plan ID and digest from canonical content.
func finalizePlanDigest(plan *Plan) {
	parts := []string{
		SchemaVersion, StrategyVersion, plan.AnalysisDigest, plan.ContractDigest,
		plan.BuildConfig.GOOS, plan.BuildConfig.GOARCH, strings.Join(sortedCopy(plan.BuildConfig.Tags), ","),
	}
	trs := append([]Transformation(nil), plan.Transformations...)
	sort.Slice(trs, func(i, j int) bool { return trs[i].ID < trs[j].ID })
	for _, tr := range trs {
		parts = append(parts, tr.Digest)
	}
	fps := append([]FilePlan(nil), plan.Files...)
	sort.Slice(fps, func(i, j int) bool { return fps[i].Path < fps[j].Path })
	for _, fp := range fps {
		parts = append(parts, fp.Path, fp.TransformedDigest)
	}
	plan.Digest = hashParts(parts...)
	plan.ID = shortID("bwplan", plan.Digest)
	// Assign transformation and edit ordering deterministically.
	sort.Slice(plan.Transformations, func(i, j int) bool { return plan.Transformations[i].ID < plan.Transformations[j].ID })
	sort.Slice(plan.Skipped, func(i, j int) bool { return plan.Skipped[i].CandidateID < plan.Skipped[j].CandidateID })
	sort.Slice(plan.Diagnostics, func(i, j int) bool { return plan.Diagnostics[i].Fingerprint < plan.Diagnostics[j].Fingerprint })
}
