package adapter

import (
	"crypto/sha256"
	"encoding/hex"
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
	KeyTypes     []string
	KeyParams    []int
	JoinContract JoinCardinality
	Limits       ParameterLimits
	Projection   []Column
	Contract     ResultContract
	MissingError string
	KeyParam     int
	ColumnCount  int // projection columns (excluding the leading ordinal)
	Digest       string
}

// SynthInput carries the parsed query and the declared key SQL type.
type SynthInput struct {
	Query           ParsedQuery
	KeyType         string   // compatibility shortcut for one key component
	KeyTypes        []string // one SQL type per key component, in predicate order
	JoinCardinality JoinCardinality
	Limits          ParameterLimits
}

// JoinCardinality is the caller-supplied proof obligation for a parsed join.
// BatchWeaver never infers uniqueness from SQL text or a live database schema.
type JoinCardinality string

const (
	// JoinCardinalityAtMostOne asserts that each base row matches zero or one
	// joined row, preserving the scalar query's result cardinality.
	JoinCardinalityAtMostOne JoinCardinality = "at-most-one"
)

// SynthesizeExactKey generates an ordered, parameterized PostgreSQL batch query
// for an exact-key SELECT. It never interpolates key values; the keys are bound
// as a single typed array parameter.
func SynthesizeExactKey(in SynthInput) (SynthPlan, *Rejection) {
	q := in.Query
	keyTypes := append([]string(nil), in.KeyTypes...)
	if len(keyTypes) == 0 && in.KeyType != "" {
		keyTypes = []string{in.KeyType}
	}
	if len(q.Keys) == 0 && q.KeyColumn.Name != "" {
		q.Keys = []KeyPredicate{{Column: q.KeyColumn, Param: q.KeyParam}}
	}
	if len(keyTypes) != len(q.Keys) || len(keyTypes) == 0 {
		return SynthPlan{}, &Rejection{Code: CodeParamAmbiguous, Reason: "one key SQL type is required per key predicate", Node: fmt.Sprintf("%d key predicates, %d types", len(q.Keys), len(keyTypes))}
	}
	for i, keyType := range keyTypes {
		if !safeSQLType(keyType) {
			return SynthPlan{}, &Rejection{Code: CodeParamAmbiguous, Reason: "key SQL type must be a qualified identifier without array suffixes", Node: keyType, Offset: i}
		}
	}
	if q.Join != nil && in.JoinCardinality != JoinCardinalityAtMostOne {
		return SynthPlan{}, &Rejection{Code: CodeCardinalityUnsupported, Reason: "joined synthesis requires an explicit at-most-one cardinality contract", Node: string(in.JoinCardinality)}
	}
	limits := normalizedLimits(in.Limits)
	if len(q.Keys) > limits.MaxParameters {
		return SynthPlan{}, &Rejection{Code: CodeParamLimitExceeded, Reason: "composite key exceeds the backend parameter limit", Node: fmt.Sprintf("%d > %d", len(q.Keys), limits.MaxParameters)}
	}
	rel := q.Relation.Name
	relRef := rel
	if q.Relation.Alias != "" {
		relRef = q.Relation.Alias
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
	requestedColumns := make([]string, len(q.Keys))
	unnestArgs := make([]string, len(q.Keys))
	keyParams := make([]int, len(q.Keys))
	for i, key := range q.Keys {
		requestedColumns[i] = requestedKeyName(i, len(q.Keys))
		unnestArgs[i] = fmt.Sprintf("$%d::%s[]", key.Param, keyTypes[i])
		keyParams[i] = key.Param
	}
	fmt.Fprintf(&b, "WITH bw_requested(%s, bw_ord) AS (\n", strings.Join(requestedColumns, ", "))
	fmt.Fprintf(&b, "\tSELECT * FROM unnest(%s) WITH ORDINALITY\n", strings.Join(unnestArgs, ", "))
	fmt.Fprintf(&b, ")\n")
	fmt.Fprintf(&b, "SELECT bw_requested.bw_ord, %s\n", strings.Join(proj, ", "))
	fmt.Fprintf(&b, "FROM bw_requested\n")
	fmt.Fprintf(&b, "LEFT JOIN %s", rel)
	if q.Relation.Alias != "" {
		fmt.Fprintf(&b, " %s", q.Relation.Alias)
	}
	conditions := make([]string, len(q.Keys))
	for i, key := range q.Keys {
		keyCol := key.Column
		if keyCol.Table == "" {
			keyCol.Table = relRef
		}
		conditions[i] = columnSQL(keyCol) + " = bw_requested." + requestedColumns[i]
	}
	fmt.Fprintf(&b, " ON %s\n", strings.Join(conditions, " AND "))
	if q.Join != nil {
		joinKeyword := "INNER JOIN"
		if q.Join.Kind == JoinLeft {
			joinKeyword = "LEFT JOIN"
		}
		fmt.Fprintf(&b, "%s %s", joinKeyword, q.Join.Relation.Name)
		if q.Join.Relation.Alias != "" {
			fmt.Fprintf(&b, " %s", q.Join.Relation.Alias)
		}
		fmt.Fprintf(&b, " ON %s = %s\n", columnSQL(q.Join.Left), columnSQL(q.Join.Right))
	}
	if len(q.Extra) > 0 {
		fmt.Fprintf(&b, "WHERE %s\n", strings.Join(q.Extra, " AND "))
	}
	fmt.Fprintf(&b, "ORDER BY bw_requested.bw_ord")

	plan := SynthPlan{
		Dialect: "postgres", Query: b.String(), KeyType: keyTypes[0], KeyTypes: keyTypes, KeyParams: keyParams, JoinContract: in.JoinCardinality,
		Limits:     limits,
		Projection: q.Projection, Contract: ContractOrderedSparseMissing,
		MissingError: "sql.ErrNoRows", KeyParam: q.Keys[0].Param, ColumnCount: len(q.Projection),
	}
	plan.Digest = synthPlanDigest(plan)
	return plan, nil
}

// Validate rejects a corrupted or hand-mutated synthesis plan before it reaches
// a database. It is an integrity check for generated plans, not a SQL parser.
func (p SynthPlan) Validate() error {
	if p.Dialect != "postgres" || p.Query == "" || p.Contract != ContractOrderedSparseMissing || len(p.KeyTypes) == 0 || len(p.KeyTypes) != len(p.KeyParams) {
		return fmt.Errorf("invalid SQL synthesis plan contract")
	}
	if p.Digest == "" || p.Digest != synthPlanDigest(p) {
		return fmt.Errorf("SQL synthesis plan digest mismatch")
	}
	return nil
}

func synthPlanDigest(p SynthPlan) string {
	h := sha256.New()
	write := func(value string) { _, _ = h.Write([]byte(value)); _, _ = h.Write([]byte{0}) }
	write(p.Dialect)
	write(p.Query)
	write(string(p.Contract))
	write(p.MissingError)
	write(string(p.JoinContract))
	write(fmt.Sprintf("limits:%d:%d:%d", p.Limits.MaxItems, p.Limits.MaxParameters, p.Limits.MaxPayloadKB))
	for i := range p.KeyTypes {
		write(fmt.Sprintf("%d:%s", p.KeyParams[i], p.KeyTypes[i]))
	}
	for _, col := range p.Projection {
		write(columnSQL(col) + ":" + col.Alias)
	}
	return "sha256:" + hex.EncodeToString(h.Sum(nil))
}

func requestedKeyName(index, count int) string {
	if count == 1 {
		return "bw_key"
	}
	return fmt.Sprintf("bw_key_%d", index+1)
}

func safeSQLType(value string) bool {
	parts := strings.Split(value, ".")
	if len(parts) == 0 {
		return false
	}
	for _, part := range parts {
		if part == "" {
			return false
		}
		for i, r := range part {
			if r == '_' || r >= 'a' && r <= 'z' || r >= 'A' && r <= 'Z' || i > 0 && r >= '0' && r <= '9' {
				continue
			}
			return false
		}
	}
	return true
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

func normalizedLimits(limits ParameterLimits) ParameterLimits {
	defaults := DefaultLimits()
	if limits.MaxItems <= 0 {
		limits.MaxItems = defaults.MaxItems
	}
	if limits.MaxParameters <= 0 {
		limits.MaxParameters = defaults.MaxParameters
	}
	if limits.MaxPayloadKB <= 0 {
		limits.MaxPayloadKB = defaults.MaxPayloadKB
	}
	return limits
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
