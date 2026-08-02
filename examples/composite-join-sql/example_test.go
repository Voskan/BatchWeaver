package compositejoinsql_test

import (
	"strings"
	"testing"

	"github.com/Voskan/BatchWeaver/internal/adapter"
)

func TestCompositeJoinSynthesis(t *testing.T) {
	parsed, rejection := adapter.ParseExactKeySelect("SELECT u.tenant_id, u.id, p.display_name FROM users u LEFT JOIN profiles p ON p.user_id = u.id WHERE u.tenant_id = $1 AND u.id = $2")
	if rejection != nil {
		t.Fatal(rejection)
	}
	plan, rejection := adapter.SynthesizeExactKey(adapter.SynthInput{
		Query: parsed, KeyTypes: []string{"uuid", "bigint"},
		JoinCardinality: adapter.JoinCardinalityAtMostOne,
	})
	if rejection != nil {
		t.Fatal(rejection)
	}
	if err := plan.Validate(); err != nil {
		t.Fatal(err)
	}
	for _, required := range []string{"WITH ORDINALITY", "LEFT JOIN profiles p", "ORDER BY bw_requested.bw_ord"} {
		if !strings.Contains(plan.Query, required) {
			t.Fatalf("generated SQL missing %q:\n%s", required, plan.Query)
		}
	}
}
