package adapter

import (
	"fmt"
	"strings"
)

// ResultContract describes how a synthesized batch maps to scalar outcomes.
type ResultContract string

const (
	// ContractOrderedSparseMissing maps one outcome per request ordinal, with a
	// declared missing outcome (typically sql.ErrNoRows) for keys with no row.
	ContractOrderedSparseMissing ResultContract = "ordered sparse-with-missing"
)

// SynthPlan is a deterministic exact-key synthesis plan.
type SynthPlan struct {
	Dialect      string
	Query        string
	KeyType      string
	Projection   []Column
	Contract     ResultContract
	MissingError string
	KeyParam     int
	ColumnCount  int // projection columns (excluding the leading ordinal)
}

// SynthInput carries the parsed query and the declared key SQL type.
type SynthInput struct {
	Query   ParsedQuery
	KeyType string // e.g. "bigint", "text", "uuid"; from the operation contract
}

// SynthesizeExactKey generates an ordered, parameterized PostgreSQL batch query
// for an exact-key SELECT. It never interpolates key values; the keys are bound
// as a single typed array parameter.
func SynthesizeExactKey(in SynthInput) (SynthPlan, *Rejection) {
	q := in.Query
	if in.KeyType == "" {
		return SynthPlan{}, &Rejection{Code: CodeParamAmbiguous, Reason: "key SQL type is required for array synthesis", Node: columnSQL(q.KeyColumn)}
	}
	rel := q.Relation.Name
	relRef := rel
	if q.Relation.Alias != "" {
		relRef = q.Relation.Alias
	}
	// The key column, qualified to the relation reference when unqualified.
	keyCol := q.KeyColumn
	if keyCol.Table == "" {
		keyCol.Table = relRef
	}

	var proj []string
	for _, c := range q.Projection {
		cc := c
		if cc.Table == "" {
			cc.Table = relRef
		}
		proj = append(proj, columnSQL(cc))
	}

	var b strings.Builder
	fmt.Fprintf(&b, "WITH bw_requested(bw_key, bw_ord) AS (\n")
	fmt.Fprintf(&b, "\tSELECT * FROM unnest($%d::%s[]) WITH ORDINALITY\n", q.KeyParam, in.KeyType)
	fmt.Fprintf(&b, ")\n")
	fmt.Fprintf(&b, "SELECT bw_requested.bw_ord, %s\n", strings.Join(proj, ", "))
	fmt.Fprintf(&b, "FROM bw_requested\n")
	fmt.Fprintf(&b, "LEFT JOIN %s", rel)
	if q.Relation.Alias != "" {
		fmt.Fprintf(&b, " %s", q.Relation.Alias)
	}
	fmt.Fprintf(&b, " ON %s = bw_requested.bw_key\n", columnSQL(keyCol))
	if len(q.Extra) > 0 {
		fmt.Fprintf(&b, "WHERE %s\n", strings.Join(q.Extra, " AND "))
	}
	fmt.Fprintf(&b, "ORDER BY bw_requested.bw_ord")

	return SynthPlan{
		Dialect: "postgres", Query: b.String(), KeyType: in.KeyType,
		Projection: q.Projection, Contract: ContractOrderedSparseMissing,
		MissingError: "sql.ErrNoRows", KeyParam: q.KeyParam, ColumnCount: len(q.Projection),
	}, nil
}

// ParameterLimits bounds a single batch to respect backend and driver limits.
type ParameterLimits struct {
	MaxItems      int
	MaxParameters int
	MaxPayloadKB  int
}

// DefaultLimits returns conservative PostgreSQL-oriented limits.
func DefaultLimits() ParameterLimits {
	return ParameterLimits{MaxItems: 1000, MaxParameters: 32000, MaxPayloadKB: 4096}
}

// Chunks splits n items into deterministic contiguous [start,end) ranges that
// respect MaxItems, preserving order and duplicates.
func Chunks(n int, limits ParameterLimits) [][2]int {
	size := limits.MaxItems
	if size <= 0 {
		size = 1000
	}
	var out [][2]int
	for start := 0; start < n; start += size {
		end := start + size
		if end > n {
			end = n
		}
		out = append(out, [2]int{start, end})
	}
	if n == 0 {
		return nil
	}
	return out
}
