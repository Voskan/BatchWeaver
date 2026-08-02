package adapter

import "testing"

func FuzzSQLSynthesis(f *testing.F) {
	for _, seed := range []string{
		"SELECT id FROM users WHERE id = $1",
		"SELECT u.tenant_id, u.id, p.name FROM users u LEFT JOIN profiles p ON p.user_id = u.id WHERE u.tenant_id = $1 AND u.id = $2",
		"SELECT * FROM users WHERE id = $1",
		"UPDATE users SET name = $1 WHERE id = $2",
	} {
		f.Add(seed, "bigint", "uuid")
	}
	f.Fuzz(func(t *testing.T, query, firstType, secondType string) {
		if len(query) > 1<<20 || len(firstType) > 256 || len(secondType) > 256 {
			t.Skip()
		}
		parsed, rejection := ParseExactKeySelect(query)
		if rejection != nil {
			return
		}
		types := []string{firstType}
		if len(parsed.Keys) > 1 {
			types = append(types, secondType)
		}
		if len(types) > len(parsed.Keys) {
			types = types[:len(parsed.Keys)]
		}
		cardinality := JoinCardinality("")
		if parsed.Join != nil {
			cardinality = JoinCardinalityAtMostOne
		}
		plan, rejection := SynthesizeExactKey(SynthInput{
			Query: parsed, KeyTypes: types, JoinCardinality: cardinality,
		})
		if rejection != nil {
			return
		}
		if err := plan.Validate(); err != nil {
			t.Fatalf("generated plan failed integrity validation: %v", err)
		}
		again, rejection := SynthesizeExactKey(SynthInput{
			Query: parsed, KeyTypes: types, JoinCardinality: cardinality,
		})
		if rejection != nil || again.Digest != plan.Digest || again.Query != plan.Query {
			t.Fatal("SQL synthesis is not deterministic")
		}
	})
}
