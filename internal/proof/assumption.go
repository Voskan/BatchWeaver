package proof

import "sort"

// Assumption fact identifiers. An assumption may only satisfy the obligation
// types that reference the fact it declares; a fact never satisfies unrelated
// obligations.
const (
	FactSideEffectFreeRead = "side_effect_free_read"
	FactPanicFree          = "panic_free"
	FactReceiverImmutable  = "receiver_immutable"
	FactSyncEquivalent     = "synchronization_equivalent"
	FactResultFreshPerCall = "result_fresh_per_call"
)

// builtinRaceFree is the always-present, always-applied assumption that the
// analyzed program is free of data races. It is recorded in every certificate
// whose eligibility depends on shared-memory reasoning, per the Go memory model
// trust boundary.
const builtinRaceFree = "BW-A-RACEFREE"

// Assumption is an explicit, scoped fact accepted from a contract or user
// assertion rather than inferred. Assumptions are never applied automatically
// beyond the built-in race-free prerequisite.
type Assumption struct {
	// Symbol is the fully qualified symbol the assumption is scoped to, for
	// example "example.com/p/internal/client.(*Client).Get". An empty symbol is
	// a workspace-wide wildcard and is rejected unless unsafe mode is enabled.
	Symbol string
	// Facts is the set of declared facts (see Fact* constants).
	Facts []string
	// Origin describes where the assumption came from (e.g. "config", "flag").
	Origin string
	// Rationale is the human-provided justification.
	Rationale string
}

// assumptionSet indexes user assumptions by symbol and tracks usage.
type assumptionSet struct {
	bySymbol map[string]Assumption
	// requested counts how many candidates asked for an assumption keyed by a
	// synthetic requirement ID.
	requested map[string]*requestedAssumption
	order     []string
	// unsafeWildcards indicates a wildcard assertion was accepted in unsafe mode.
	unsafeWildcards bool
}

// requestedAssumption records an assumption the engine needs but that has not
// been supplied.
type requestedAssumption struct {
	id     string
	text   string
	symbol string
	facts  []string
	count  int
}

// newAssumptionSet indexes the supplied assumptions.
func newAssumptionSet(as []Assumption) *assumptionSet {
	s := &assumptionSet{
		bySymbol:  make(map[string]Assumption, len(as)),
		requested: make(map[string]*requestedAssumption),
	}
	for _, a := range as {
		if a.Symbol == "" {
			s.unsafeWildcards = true
			continue
		}
		s.bySymbol[a.Symbol] = a
	}
	return s
}

// has reports whether a symbol was asserted with the given fact.
func (s *assumptionSet) has(symbol, fact string) bool {
	a, ok := s.bySymbol[symbol]
	if !ok {
		return false
	}
	for _, f := range a.Facts {
		if f == fact {
			return true
		}
	}
	return false
}

// digest returns a stable digest of the applied assumption set. It contributes
// to certificate identity and invalidation.
func (s *assumptionSet) digest() string {
	if s == nil || len(s.bySymbol) == 0 {
		return ""
	}
	syms := make([]string, 0, len(s.bySymbol))
	for k := range s.bySymbol {
		syms = append(syms, k)
	}
	sort.Strings(syms)
	var parts []string
	for _, k := range syms {
		a := s.bySymbol[k]
		parts = append(parts, k)
		parts = append(parts, sortedCopy(a.Facts)...)
	}
	return hashParts(parts...)
}

// request records that a candidate needs an assumption about symbol/fact that
// was not supplied, and returns its stable requirement ID.
func (s *assumptionSet) request(symbol, fact, text string) string {
	id := shortID("BW-A", symbol, fact)
	r, ok := s.requested[id]
	if !ok {
		r = &requestedAssumption{id: id, text: text, symbol: symbol, facts: []string{fact}}
		s.requested[id] = r
		s.order = append(s.order, id)
	}
	r.count++
	return id
}

// requestedRefs returns the requested assumptions as report references, sorted
// by ID.
func (s *assumptionSet) requestedRefs() []AssumptionRef {
	if s == nil {
		return nil
	}
	ids := append([]string(nil), s.order...)
	sort.Strings(ids)
	out := make([]AssumptionRef, 0, len(ids))
	for _, id := range ids {
		r := s.requested[id]
		out = append(out, AssumptionRef{
			ID:               r.id,
			Text:             r.text,
			Origin:           "inferred-requirement",
			Symbol:           r.symbol,
			Facts:            r.facts,
			Digest:           hashParts(r.symbol, hashParts(r.facts...)),
			RequestedByCount: r.count,
			Applied:          false,
		})
	}
	return out
}
