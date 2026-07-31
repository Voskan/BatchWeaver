package transform

import (
	"bytes"
	"context"
	"fmt"
	"go/format"
	"path/filepath"
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

	report, snap, syms, contractDigest, err := runProof(ctx, req, patterns)
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

	wantStatic := containsStrategy(strategies, StrategyStaticLoopPrefetch)
	wantRuntime := false
	for _, s := range strategies {
		if RuntimeStrategies(s) {
			wantRuntime = true
		}
	}

	candLocs := candidateLocations(snap)

	edits := map[string][]pendingEdit{} // rel -> edits
	fileMode := map[string]string{}     // rel -> "static" | "runtime"
	bridges := map[string]*bridgeReq{}  // bridge file rel -> req
	created := map[string][]byte{}      // bridge file rel -> content
	count := 0

	for _, cp := range report.CandidateProofs {
		if req.Filter.Max > 0 && count >= req.Filter.Max {
			break
		}
		if !matchesFilter(req.Filter, cp) {
			continue
		}
		cert := newCertificate(cp)
		sym, hasSym := syms[cp.Operation]

		// Static loop prefetch takes precedence when requested and applicable.
		if wantStatic && cert.eligibleForProof(proof.StrategyStaticLoopPrefetch) {
			if !hasSym || sym.scalar == "" || sym.batch == "" {
				plan.Skipped = append(plan.Skipped, skip(cp, SkipBindingUnavailable, "operation has no resolved scalar/batch symbols"))
				continue
			}
			site, reason := l.locateLoop(cert.Location, sym.scalar, sym.batch)
			if reason == "" {
				if fileMode[site.fi.rel] != "" {
					plan.Skipped = append(plan.Skipped, skip(cp, SkipOverlappingRegion, "another transformation already targets this file"))
					continue
				}
				_, replacement, startOff, endOff := l.generate(site)
				edits[site.fi.rel] = append(edits[site.fi.rel], pendingEdit{rel: site.fi.rel, start: startOff, end: endOff, repl: replacement})
				fileMode[site.fi.rel] = "static"
				plan.Transformations = append(plan.Transformations, buildStaticTransformation(cp, cert, site, l, startOff, endOff, replacement))
				count++
				continue
			}
			if !wantRuntime || !runtimeEligible(cert) {
				plan.Skipped = append(plan.Skipped, skip(cp, reason, ""))
				continue
			}
			// Fall through to runtime lowering.
		}

		// Runtime bridge lowering (standalone, sibling, loop, or fan-out).
		if wantRuntime && runtimeEligible(cert) {
			if !hasSym || sym.scalar == "" {
				plan.Skipped = append(plan.Skipped, skip(cp, SkipBindingUnavailable, "operation has no resolved scalar symbol"))
				continue
			}
			re, br, reason := l.lowerRuntime(cp.Operation, candLocs[cp.ID], sym.scalar)
			if reason != "" {
				plan.Skipped = append(plan.Skipped, skip(cp, reason, ""))
				continue
			}
			conflict := false
			for _, e := range re {
				if fileMode[e.rel] == "static" {
					conflict = true
				}
			}
			if conflict {
				plan.Skipped = append(plan.Skipped, skip(cp, SkipOverlappingRegion, "a static transformation already targets this file"))
				continue
			}
			for _, e := range re {
				edits[e.rel] = append(edits[e.rel], e)
				fileMode[e.rel] = "runtime"
			}
			bridges[br.file] = br
			plan.Transformations = append(plan.Transformations, buildRuntimeTransformation(cp, cert, br, re, runtimeStrategyFor(cp, strategies)))
			count++
			continue
		}

		reason := SkipNotEligible
		if !wantStatic && !wantRuntime {
			reason = SkipStrategyNotRequested
		}
		plan.Skipped = append(plan.Skipped, skip(cp, reason, string(cp.Decision)))
	}

	// Generate bridge files for runtime lowerings.
	for rel, br := range bridges {
		content, gerr := generateBridge(br)
		if gerr != nil {
			plan.Diagnostics = append(plan.Diagnostics, diag("BW3401", "error", "generated bridge does not format", rel, "", gerr.Error()))
			continue
		}
		created[rel] = content
	}

	buildFilePlans(plan, l, edits, created)

	if len(plan.Files) > 0 {
		plan.Validation.Parse = ValidationPassed
		plan.Validation.Structural = structuralOK(plan)
		if err := typeCheckOverlay(ctx, l, plan); err != nil {
			plan.Validation.TypeCheck = ValidationFailed
			plan.Validation.Detail = err.Error()
			plan.Diagnostics = append(plan.Diagnostics, diag("BW3402", "error", "transformed package does not type-check", "", "", err.Error()))
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

// buildFilePlans applies edits to modified files and adds created bridge files.
func buildFilePlans(plan *Plan, l *loaded, edits map[string][]pendingEdit, created map[string][]byte) {
	var modRels []string
	for rel := range edits {
		modRels = append(modRels, rel)
	}
	sort.Strings(modRels)
	for _, rel := range modRels {
		fi := l.byFile[rel]
		if fi == nil {
			continue
		}
		transformed, aerr := applyEdits(fi.src, edits[rel])
		if aerr != nil {
			plan.Diagnostics = append(plan.Diagnostics, diag("BW3301", "error", "edits overlap", rel, "", aerr.Error()))
			continue
		}
		formatted, ferr := format.Source(transformed)
		if ferr != nil {
			plan.Diagnostics = append(plan.Diagnostics, diag("BW3401", "error", "transformed file does not format", rel, "", ferr.Error()))
			continue
		}
		ins, rem := lineDelta(fi.src, formatted)
		plan.Files = append(plan.Files, FilePlan{
			Path: rel, OriginalDigest: hashBytes(fi.src), TransformedDigest: hashBytes(formatted),
			InsertedLines: ins, RemovedLines: rem, transformed: formatted, original: fi.src,
		})
	}
	var crRels []string
	for rel := range created {
		crRels = append(crRels, rel)
	}
	sort.Strings(crRels)
	for _, rel := range crRels {
		content := created[rel]
		plan.Files = append(plan.Files, FilePlan{
			Path: rel, OriginalDigest: hashBytes(nil), TransformedDigest: hashBytes(content),
			InsertedLines: bytes.Count(content, []byte{'\n'}), Created: true, Generated: true,
			transformed: content, original: nil,
		})
	}
	sort.Slice(plan.Files, func(i, j int) bool { return plan.Files[i].Path < plan.Files[j].Path })
}

// applyEdits applies non-overlapping edits to src, left to right.
func applyEdits(src []byte, edits []pendingEdit) ([]byte, error) {
	es := append([]pendingEdit(nil), edits...)
	sort.Slice(es, func(i, j int) bool { return es[i].start < es[j].start })
	for i := 1; i < len(es); i++ {
		if es[i].start < es[i-1].end {
			return nil, fmt.Errorf("edit at %d overlaps edit ending at %d", es[i].start, es[i-1].end)
		}
	}
	out := make([]byte, 0, len(src)+64)
	prev := 0
	for _, e := range es {
		out = append(out, src[prev:e.start]...)
		out = append(out, e.repl...)
		prev = e.end
	}
	out = append(out, src[prev:]...)
	return out, nil
}

// candidateLocations maps candidate ID to its call-site locations.
func candidateLocations(snap *analysis.Snapshot) map[string][]siteLoc {
	siteByID := make(map[string]analysis.CallSite, len(snap.CallSites))
	for _, s := range snap.CallSites {
		siteByID[s.ID] = s
	}
	out := map[string][]siteLoc{}
	for _, c := range snap.Candidates {
		for _, id := range c.CallSites {
			if s, ok := siteByID[id]; ok {
				if rel, line, _, ok := parseLocation(s.Location); ok {
					out[c.ID] = append(out[c.ID], siteLoc{rel: rel, line: line})
				}
			}
		}
	}
	return out
}

func containsStrategy(list []StrategyID, want StrategyID) bool {
	for _, s := range list {
		if s == want {
			return true
		}
	}
	return false
}

// runtimeEligible reports whether a certificate proves any runtime-lowering
// strategy eligible.
func runtimeEligible(cert Certificate) bool {
	return cert.eligibleForProof(proof.StrategyRuntimeScopeCoalescing) ||
		cert.eligibleForProof(proof.StrategyExistingFanoutCoalescing)
}

// runtimeStrategyFor selects the transformation strategy label for a runtime
// lowering, preferring the most specific strategy the user requested for the
// candidate's structure. It never labels a lowering with a strategy the user did
// not request.
func runtimeStrategyFor(cp proof.CandidateProof, requested []StrategyID) StrategyID {
	has := func(s StrategyID) bool { return containsStrategy(requested, s) }
	if strings.Contains(cp.Structure, "fan-out") {
		if has(StrategyFanoutCoalescing) {
			return StrategyFanoutCoalescing
		}
		if has(StrategyErrgroupCoalescing) {
			return StrategyErrgroupCoalescing
		}
	}
	if strings.Contains(cp.Structure, "sibling") && has(StrategyStaticSiblingFusion) {
		return StrategyStaticSiblingFusion
	}
	if has(StrategyRuntimeCallCoalescing) {
		return StrategyRuntimeCallCoalescing
	}
	for _, s := range requested {
		if RuntimeStrategies(s) {
			return s
		}
	}
	return StrategyRuntimeCallCoalescing
}

// runProof runs analysis and proof and resolves operation symbols. It also
// returns the analysis snapshot so callers can enumerate call-site locations.
func runProof(ctx context.Context, req Request, patterns []string) (*proof.Report, *analysis.Snapshot, map[string]symbols, string, error) {
	snap, err := analysis.Analyze(ctx, analysis.Request{
		Patterns: patterns, Reproducible: true, ToolVersion: req.ToolVersion, Dir: req.Dir,
		BuildContext: analysis.BuildContext{
			GOOS: req.BuildConfig.GOOS, GOARCH: req.BuildConfig.GOARCH,
			Tags: req.BuildConfig.Tags, Tests: req.BuildConfig.Tests,
		},
	})
	if err != nil {
		return nil, nil, nil, "", err
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
		return nil, nil, nil, "", err
	}
	return report, snap, syms, contractDigest, nil
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

// buildStaticTransformation constructs the IR record for one static loop prefetch.
func buildStaticTransformation(cp proof.CandidateProof, cert Certificate, site *loopSite, l *loaded, startOff, endOff int, replacement string) Transformation {
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
		OriginalDigest: hashBytes(site.fi.src[startOff:endOff]),
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

// buildRuntimeTransformation constructs the IR record for one runtime lowering.
func buildRuntimeTransformation(cp proof.CandidateProof, cert Certificate, br *bridgeReq, edits []pendingEdit, strategy StrategyID) Transformation {
	anchor := SourceAnchor{
		File: edits[0].rel, Package: br.pkgName, Function: "",
		StructuralHash: hashParts("runtime", cp.Operation, br.varName, fmt.Sprint(len(edits)))[:shortLen],
		Resolution:     AnchorExact,
	}
	var editIDs []string
	var replParts []string
	for i, e := range edits {
		editIDs = append(editIDs, shortID("bwedit", e.rel, cp.ID, fmt.Sprint(i)))
		replParts = append(replParts, e.repl)
	}
	tr := Transformation{
		CandidateID: cp.ID, CertificateID: cp.ProofID, Strategy: strategy,
		Operation: cp.Operation, Source: anchor,
		Phases:           []Phase{PhaseBindInvariants, PhaseInvokeBatch, PhaseMapResults, PhaseFinalize},
		GeneratedSymbols: []string{br.varName},
		Edits:            editIDs,
		Assumptions:      cert.Assumptions,
		NonGuarantees:    cert.NonGuarantees,
		Bridge:           br.file,
		RuntimeABI:       RuntimeABIVersion,
	}
	tr.ID = shortID("bwtransform", cp.ID, cp.ProofID, string(tr.Strategy))
	tr.Digest = hashParts(tr.ID, tr.CandidateID, tr.CertificateID, string(tr.Strategy),
		anchor.StructuralHash, br.varName, RuntimeABIVersion, hashParts(replParts...))
	return tr
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

// structuralOK performs light structural verification: static prefetch must
// reference its four generated symbols; runtime lowering must reference its
// bridge symbol.
func structuralOK(plan *Plan) ValidationState {
	for _, tr := range plan.Transformations {
		if tr.Strategy == StrategyStaticLoopPrefetch {
			if len(tr.GeneratedSymbols) != 4 {
				return ValidationFailed
			}
			continue
		}
		if len(tr.GeneratedSymbols) < 1 || tr.Bridge == "" {
			return ValidationFailed
		}
	}
	return ValidationPassed
}

// typeCheckOverlay type-checks the affected packages against an in-memory
// overlay of the transformed and generated files.
func typeCheckOverlay(ctx context.Context, l *loaded, plan *Plan) error {
	overlay := map[string][]byte{}
	pkgPaths := map[string]bool{}
	for _, fp := range plan.Files {
		if fp.Created {
			overlay[filepath.Join(l.root, filepath.FromSlash(fp.Path))] = fp.transformed
			continue
		}
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
		SchemaVersion, StrategyVersion, RuntimeABIVersion, plan.AnalysisDigest, plan.ContractDigest,
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
	sort.Slice(plan.Transformations, func(i, j int) bool { return plan.Transformations[i].ID < plan.Transformations[j].ID })
	sort.Slice(plan.Skipped, func(i, j int) bool { return plan.Skipped[i].CandidateID < plan.Skipped[j].CandidateID })
	sort.Slice(plan.Diagnostics, func(i, j int) bool { return plan.Diagnostics[i].Fingerprint < plan.Diagnostics[j].Fingerprint })
}
