package transform

import (
	"sort"
	"strings"
)

// SourceMap is a versioned map from generated code back to candidates and roles.
type SourceMap struct {
	SchemaVersion string             `json:"schema_version"`
	PlanID        string             `json:"plan_id"`
	Segments      []SourceMapSegment `json:"segments"`
}

// SourceMapSchema versions the source-map artifact.
const SourceMapSchema = "batchweaver.sourcemap/v1alpha1"

// BuildSourceMap builds a source map for a plan. Segments are located by scanning
// each transformed file for the generated identifiers of each transformation.
// The mapping is line-granular and deterministic; sub-line precision is a
// documented limitation of this stage.
func BuildSourceMap(plan *Plan) SourceMap {
	sm := SourceMap{SchemaVersion: SourceMapSchema, PlanID: plan.ID}
	byFile := map[string][]Transformation{}
	for _, tr := range plan.Transformations {
		byFile[tr.Source.File] = append(byFile[tr.Source.File], tr)
	}
	for _, fp := range plan.Files {
		lines := strings.Split(string(fp.transformed), "\n")
		for _, tr := range byFile[fp.Path] {
			sm.Segments = append(sm.Segments, locateRoles(fp.Path, lines, tr)...)
		}
	}
	sort.Slice(sm.Segments, func(i, j int) bool {
		if sm.Segments[i].File != sm.Segments[j].File {
			return sm.Segments[i].File < sm.Segments[j].File
		}
		return sm.Segments[i].GeneratedStart < sm.Segments[j].GeneratedStart
	})
	return sm
}

// locateRoles finds representative generated lines for a transformation's roles.
func locateRoles(file string, lines []string, tr Transformation) []SourceMapSegment {
	if tr.Strategy == StrategyExactKeySQLSynthesis || tr.Strategy == StrategyCompositeKeySQLSynthesis || tr.Strategy == StrategyBoundedJoinSQLSynthesis {
		if len(tr.GeneratedSymbols) == 0 {
			return nil
		}
		for i, line := range lines {
			if strings.Contains(line, "const "+tr.GeneratedSymbols[0]+" =") {
				return []SourceMapSegment{{
					ID: shortID("bwseg", file, string(RoleSQLSynthesis), tr.ID), File: file,
					GeneratedStart: i + 1, GeneratedEnd: i + 1, Role: RoleSQLSynthesis,
					Transformation: tr.ID, Candidate: tr.CandidateID, Certificate: tr.CertificateID,
				}}
			}
		}
		return nil
	}
	if len(tr.GeneratedSymbols) < 4 {
		return nil
	}
	keys, values, _, index := tr.GeneratedSymbols[0], tr.GeneratedSymbols[1], tr.GeneratedSymbols[2], tr.GeneratedSymbols[3]
	find := func(substr string) int {
		for i, ln := range lines {
			if strings.Contains(ln, substr) {
				return i + 1
			}
		}
		return 0
	}
	seg := func(role GeneratedRole, line int) SourceMapSegment {
		return SourceMapSegment{
			ID: shortID("bwseg", file, string(role), tr.ID), File: file,
			GeneratedStart: line, GeneratedEnd: line, Role: role,
			Transformation: tr.ID, Candidate: tr.CandidateID, Certificate: tr.CertificateID,
		}
	}
	var out []SourceMapSegment
	if l := find(keys + " := make("); l > 0 {
		out = append(out, seg(RoleKeyCollection, l))
	}
	if l := find(values + ", "); l > 0 {
		out = append(out, seg(RoleBatchCall, l))
	}
	if l := find(index + " := 0"); l > 0 {
		out = append(out, seg(RoleResultRecon, l))
	}
	return out
}
