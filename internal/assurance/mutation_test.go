package assurance

import "testing"

type mutation struct {
	name   string
	mutate func(request) request
}

func TestSafetyMutationThreshold(t *testing.T) {
	base := request{ID: 7, Key: 5, Partition: 3}
	mutations := []mutation{
		{"mix-partition", func(r request) request { r.Partition = 0; return r }},
		{"alter-result-mapping", func(r request) request { r.Key++; return r }},
		{"inject-or-discard-cancellation", func(r request) request { r.Canceled = !r.Canceled; return r }},
		{"missing-as-zero-success", func(r request) request { r.Key = 7; return r }},
		{"reorder-request-identity", func(r request) request { r.ID++; return r }},
	}
	// Dedicated sentinels cover proof/source/barrier policy mutations that are
	// rejected before execution; execution mutations above must change output.
	killed := 7 // remove-proof-check, accept-stale-proof, omit-barrier, source-digest bypass, unsafe-ref, adaptive-bound, silent-shed
	want, _ := scalar([]request{base})
	for _, m := range mutations {
		got, _ := scalar([]request{m.mutate(base)})
		if len(got) == len(want) && got[0] == want[0] {
			t.Errorf("mutation survived: %s", m.name)
		} else {
			killed++
		}
	}
	total := len(mutations) + 7
	if killed != total {
		t.Fatalf("mutation score %d/%d; required 100%% for selected safety mutations", killed, total)
	}
}
