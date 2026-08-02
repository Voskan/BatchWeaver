package adapter

import (
	"context"
	"database/sql"
	"database/sql/driver"
	"errors"
	"io"
	"strings"
	"testing"

	batchweaver "github.com/Voskan/BatchWeaver"
)

func TestManifestsValidAndDeterministic(t *testing.T) {
	t.Parallel()
	ms := Manifests()
	if len(ms) == 0 {
		t.Fatal("no manifests")
	}
	for _, m := range ms {
		if err := m.Validate(); err != nil {
			t.Errorf("manifest %s invalid: %v", m.AdapterID, err)
		}
		if m.Digest == "" {
			t.Errorf("manifest %s has no digest", m.AdapterID)
		}
	}
	// Determinism: recompute.
	if Manifests()[0].Digest != ms[0].Digest {
		t.Error("manifest digest is not deterministic")
	}
	databaseSQL, ok := ManifestByID("database/sql")
	if !ok {
		t.Error("database/sql manifest missing")
	} else if !databaseSQL.HasCapability(CapCompositeKeyRead) || !databaseSQL.HasCapability(CapBoundedJoinRead) {
		t.Error("database/sql manifest does not advertise implemented composite/join synthesis")
	}
	if pgx, ok := ManifestByID("pgx"); !ok || pgx.HasCapability(CapBoundedJoinRead) {
		t.Error("pgx manifest overclaims database/sql bounded-join synthesis")
	}
}

func TestParseAccept(t *testing.T) {
	t.Parallel()
	q, rej := ParseExactKeySelect("SELECT id, name FROM users WHERE id = $1")
	if rej != nil {
		t.Fatalf("unexpected rejection: %v", rej)
	}
	if q.Relation.Name != "users" || q.KeyColumn.Name != "id" || q.KeyParam != 1 || len(q.Projection) != 2 {
		t.Fatalf("bad parse: %+v", q)
	}
	q2, rej := ParseExactKeySelect("SELECT id, name FROM users u WHERE u.id = $1 AND deleted_at IS NULL")
	if rej != nil {
		t.Fatalf("unexpected rejection: %v", rej)
	}
	if len(q2.Extra) != 1 || !strings.Contains(q2.Extra[0], "IS NULL") {
		t.Fatalf("extra predicate not captured: %+v", q2.Extra)
	}
}

func TestParseReject(t *testing.T) {
	t.Parallel()
	cases := []struct {
		sql  string
		code string
	}{
		{"SELECT * FROM users WHERE id = $1", CodeProjectionAmbiguous},
		{"SELECT id, now() FROM users WHERE id = $1", CodeSQLVolatile},
		{"SELECT a.id FROM a RIGHT JOIN b ON a.x = b.x WHERE a.id = $1", CodeSQLUnsupported},
		{"INSERT INTO users VALUES ($1)", CodeSQLUnsupported},
		{"SELECT id FROM users WHERE id = $1 ORDER BY id", CodeSQLUnsupported},
		{"SELECT id FROM users WHERE id = $1 LIMIT 1", CodeSQLUnsupported},
		{"SELECT id FROM users WHERE id = $1 FOR UPDATE", CodeSQLUnsupported},
		{"SELECT id FROM users WHERE id = 5", CodeParamAmbiguous},
		{"SELECT id FROM users WHERE id = $1; DROP TABLE users", CodeSQLUnsupported},
		{"SELECT id FROM users WHERE id = $1 OR name = $2", CodeSQLUnsupported},
	}
	for _, c := range cases {
		_, rej := ParseExactKeySelect(c.sql)
		if rej == nil {
			t.Errorf("expected rejection for %q", c.sql)
			continue
		}
		if rej.Code != c.code {
			t.Errorf("%q: code = %s, want %s (%s)", c.sql, rej.Code, c.code, rej.Reason)
		}
	}
}

func TestParseNeverPanics(t *testing.T) {
	t.Parallel()
	for _, s := range []string{"", "   ", ";;;", "SELECT", "SELECT (", "$", "\"unterminated", "'unterminated", "SELECT a.", "/* c"} {
		_, _ = ParseExactKeySelect(s) // must not panic
	}
}

func TestSynthesizeExactKey(t *testing.T) {
	t.Parallel()
	q, rej := ParseExactKeySelect("SELECT id, name FROM users WHERE id = $1")
	if rej != nil {
		t.Fatal(rej)
	}
	plan, rej := SynthesizeExactKey(SynthInput{Query: q, KeyType: "bigint"})
	if rej != nil {
		t.Fatal(rej)
	}
	for _, want := range []string{
		"unnest($1::bigint[]) WITH ORDINALITY",
		"LEFT JOIN users",
		"ON users.id = bw_requested.bw_key",
		"ORDER BY bw_requested.bw_ord",
	} {
		if !strings.Contains(plan.Query, want) {
			t.Errorf("generated query missing %q:\n%s", want, plan.Query)
		}
	}
	if plan.Contract != ContractOrderedSparseMissing {
		t.Errorf("contract = %s", plan.Contract)
	}
}

func TestSynthesizeCompositeKeyDeterministically(t *testing.T) {
	t.Parallel()
	q, rej := ParseExactKeySelect("SELECT tenant_id, id, name FROM users WHERE tenant_id = $1 AND id = $2")
	if rej != nil {
		t.Fatal(rej)
	}
	if len(q.Keys) != 2 || q.Keys[0].Param != 1 || q.Keys[1].Param != 2 {
		t.Fatalf("unexpected keys: %+v", q.Keys)
	}
	plan, rej := SynthesizeExactKey(SynthInput{Query: q, KeyTypes: []string{"uuid", "bigint"}})
	if rej != nil {
		t.Fatal(rej)
	}
	for _, want := range []string{
		"bw_requested(bw_key_1, bw_key_2, bw_ord)",
		"unnest($1::uuid[], $2::bigint[]) WITH ORDINALITY",
		"users.tenant_id = bw_requested.bw_key_1 AND users.id = bw_requested.bw_key_2",
	} {
		if !strings.Contains(plan.Query, want) {
			t.Errorf("generated query missing %q:\n%s", want, plan.Query)
		}
	}
	if len(plan.KeyParams) != 2 || plan.KeyParams[0] != 1 || plan.KeyParams[1] != 2 {
		t.Fatalf("unexpected key parameters: %v", plan.KeyParams)
	}
}

func TestBoundedJoinRequiresExplicitCardinality(t *testing.T) {
	t.Parallel()
	q, rej := ParseExactKeySelect("SELECT u.id, p.display_name FROM users u LEFT JOIN profiles p ON p.user_id = u.id WHERE u.id = $1")
	if rej != nil {
		t.Fatal(rej)
	}
	if q.Join == nil || q.Join.Kind != JoinLeft {
		t.Fatalf("unexpected join: %+v", q.Join)
	}
	if _, rej := SynthesizeExactKey(SynthInput{Query: q, KeyType: "bigint"}); rej == nil || rej.Code != CodeCardinalityUnsupported {
		t.Fatalf("join without cardinality contract = %v", rej)
	}
	plan, rej := SynthesizeExactKey(SynthInput{Query: q, KeyType: "bigint", JoinCardinality: JoinCardinalityAtMostOne})
	if rej != nil {
		t.Fatal(rej)
	}
	for _, want := range []string{"LEFT JOIN users u", "ON u.id = bw_requested.bw_key", "LEFT JOIN profiles p ON p.user_id = u.id"} {
		if !strings.Contains(plan.Query, want) {
			t.Errorf("generated query missing %q:\n%s", want, plan.Query)
		}
	}
}

func TestCompositeAndJoinRejectUnsafeShapes(t *testing.T) {
	t.Parallel()
	cases := []struct {
		sql  string
		code string
	}{
		{"SELECT tenant_id FROM users WHERE tenant_id = $2 AND id = $2", CodeParamAmbiguous},
		{"SELECT tenant_id FROM users WHERE tenant_id = $1 AND id = $3", CodeParamAmbiguous},
		{"SELECT id FROM users u JOIN profiles p ON p.user_id = u.id WHERE u.id = $1", CodeProjectionAmbiguous},
		{"SELECT u.id FROM users u JOIN profiles p ON p.user_id = p.id WHERE u.id = $1", CodeJoinUnsupported},
		{"SELECT p.id FROM users u JOIN profiles p ON p.user_id = u.id WHERE p.id = $1", CodeJoinUnsupported},
		{"SELECT u.id FROM users u JOIN profiles p ON p.user_id = u.id WHERE u.id = $1 AND other.deleted_at IS NULL", CodeJoinUnsupported},
	}
	for _, tc := range cases {
		_, rej := ParseExactKeySelect(tc.sql)
		if rej == nil || rej.Code != tc.code {
			t.Errorf("%q: rejection = %v, want %s", tc.sql, rej, tc.code)
		}
	}
	q, _ := ParseExactKeySelect("SELECT id FROM users WHERE id = $1")
	if _, rej := SynthesizeExactKey(SynthInput{Query: q, KeyType: "bigint[];DROP"}); rej == nil || rej.Code != CodeParamAmbiguous {
		t.Fatalf("unsafe SQL type accepted: %v", rej)
	}
}

func TestSynthPlanIntegrityKillsSemanticMutations(t *testing.T) {
	t.Parallel()
	plan := planFor(t, "SELECT id, name FROM users WHERE id = $1", "bigint")
	if err := plan.Validate(); err != nil {
		t.Fatal(err)
	}
	mutations := []func(*SynthPlan){
		func(p *SynthPlan) { p.Query = strings.ReplaceAll(p.Query, " WITH ORDINALITY", "") },
		func(p *SynthPlan) { p.Query = strings.ReplaceAll(p.Query, "ORDER BY bw_requested.bw_ord", "") },
		func(p *SynthPlan) { p.Contract = "unordered" },
		func(p *SynthPlan) { p.KeyTypes[0] = "text" },
	}
	for i, mutate := range mutations {
		candidate := plan
		candidate.KeyTypes = append([]string(nil), plan.KeyTypes...)
		mutate(&candidate)
		if err := candidate.Validate(); err == nil {
			t.Errorf("mutation %d survived plan validation", i)
		}
	}
}

func TestSynthesisEnforcesBackendParameterLimit(t *testing.T) {
	t.Parallel()
	q, rejection := ParseExactKeySelect("SELECT tenant_id, id FROM users WHERE tenant_id = $1 AND id = $2")
	if rejection != nil {
		t.Fatal(rejection)
	}
	_, rejection = SynthesizeExactKey(SynthInput{Query: q, KeyTypes: []string{"uuid", "bigint"}, Limits: ParameterLimits{MaxItems: 10, MaxParameters: 1, MaxPayloadKB: 1}})
	if rejection == nil || rejection.Code != CodeParamLimitExceeded {
		t.Fatalf("parameter limit rejection = %v", rejection)
	}
}

func TestRedisSlots(t *testing.T) {
	t.Parallel()
	if got := crc16([]byte("123456789")); got != 0x31C3 {
		t.Errorf("crc16 = %#x, want 0x31C3", got)
	}
	if Slot("{user1000}.following") != Slot("{user1000}.followers") {
		t.Error("hash-tagged keys must share a slot")
	}
	if Slot("{user1000}.following") != Slot("user1000") {
		t.Error("hash tag must hash only the tag content")
	}
	if SameSlot([]string{"{u}.a", "{u}.b"}) != true {
		t.Error("same-tag keys should be same slot")
	}
	groups := SlotGroups([]string{"{a}.1", "{b}.1", "{a}.2"})
	if len(groups) != 2 {
		t.Fatalf("groups = %d, want 2", len(groups))
	}
}

// --- fake database/sql driver for hermetic provider tests ---

var (
	fakeCols = []string{"bw_ord", "name"}
	fakeData [][]driver.Value
	fakeErr  error
	fakeArgs []driver.NamedValue
)

type fakeDriver struct{}

func (fakeDriver) Open(string) (driver.Conn, error) { return &fakeConn{}, nil }

type fakeConn struct{}

func (c *fakeConn) Prepare(string) (driver.Stmt, error) {
	return nil, errors.New("prepare unsupported")
}
func (c *fakeConn) Close() error              { return nil }
func (c *fakeConn) Begin() (driver.Tx, error) { return nil, errors.New("tx unsupported") }
func (c *fakeConn) CheckNamedValue(*driver.NamedValue) error {
	return nil // accept any argument, including a key slice
}
func (c *fakeConn) QueryContext(ctx context.Context, _ string, args []driver.NamedValue) (driver.Rows, error) {
	if err := ctx.Err(); err != nil {
		return nil, err
	}
	fakeArgs = append([]driver.NamedValue(nil), args...)
	if fakeErr != nil {
		return nil, fakeErr
	}
	return &fakeRows{cols: fakeCols, data: fakeData}, nil
}

type fakeRows struct {
	cols []string
	data [][]driver.Value
	pos  int
}

func (r *fakeRows) Columns() []string { return r.cols }
func (r *fakeRows) Close() error      { return nil }
func (r *fakeRows) Next(dest []driver.Value) error {
	if r.pos >= len(r.data) {
		return io.EOF
	}
	copy(dest, r.data[r.pos])
	r.pos++
	return nil
}

func init() { sql.Register("bwfake", fakeDriver{}) }

func TestSQLProviderMapping(t *testing.T) {
	db, err := sql.Open("bwfake", "")
	if err != nil {
		t.Fatal(err)
	}
	defer func() { _ = db.Close() }()

	// Rows for 3 requested keys: ord 1 and 3 present, ord 2 missing.
	fakeData = [][]driver.Value{{int64(1), "a"}, {int64(3), "c"}}
	fakeErr = nil

	plan := planFor(t, "SELECT id, name FROM users WHERE id = $1", "bigint")
	prov := SQLProvider[int, string]{
		DB: db, Plan: plan,
		KeyArg: func(ks []int) any { return ks },
		Decode: func(rows *sql.Rows) (int64, string, error) {
			var ord int64
			var name string
			err := rows.Scan(&ord, &name)
			return ord, name, err
		},
	}
	req := batchweaver.MustNewBatchRequest([]batchweaver.BatchItem[int]{
		batchweaver.NewBatchItem(1, 10), batchweaver.NewBatchItem(2, 20), batchweaver.NewBatchItem(3, 30),
	})
	resp, err := prov.Execute(context.Background(), req)
	if err != nil {
		t.Fatalf("execute: %v", err)
	}
	outs := resp.Outcomes()
	if len(outs) != 3 {
		t.Fatalf("outcomes = %d, want 3", len(outs))
	}
	if !outs[0].IsSuccess() || outs[0].Value != "a" {
		t.Errorf("item 0: %+v", outs[0])
	}
	if !errors.Is(outs[1].Err, sql.ErrNoRows) {
		t.Errorf("item 1 should be sql.ErrNoRows, got %+v", outs[1])
	}
	if !outs[2].IsSuccess() || outs[2].Value != "c" {
		t.Errorf("item 2: %+v", outs[2])
	}
}

func TestSQLProviderGlobalError(t *testing.T) {
	db, _ := sql.Open("bwfake", "")
	defer func() { _ = db.Close() }()
	fakeData = nil
	fakeErr = errors.New("connection reset")
	plan := planFor(t, "SELECT id, name FROM users WHERE id = $1", "bigint")
	prov := SQLProvider[int, string]{DB: db, Plan: plan, KeyArg: func(ks []int) any { return ks },
		Decode: func(rows *sql.Rows) (int64, string, error) { var o int64; var s string; return o, s, rows.Scan(&o, &s) }}
	req := batchweaver.MustNewBatchRequest([]batchweaver.BatchItem[int]{batchweaver.NewBatchItem(1, 10)})
	_, err := prov.Execute(context.Background(), req)
	if err == nil {
		t.Fatal("expected global error")
	}
	fakeErr = nil
}

func TestSQLProviderCompositeArgumentsAndCardinalityDefense(t *testing.T) {
	type compositeKey struct {
		Tenant string
		ID     int64
	}
	db, err := sql.Open("bwfake", "")
	if err != nil {
		t.Fatal(err)
	}
	defer func() { _ = db.Close() }()
	q, rej := ParseExactKeySelect("SELECT tenant_id, id, name FROM users WHERE tenant_id = $1 AND id = $2")
	if rej != nil {
		t.Fatal(rej)
	}
	plan, rej := SynthesizeExactKey(SynthInput{Query: q, KeyTypes: []string{"text", "bigint"}})
	if rej != nil {
		t.Fatal(rej)
	}
	provider := SQLProvider[compositeKey, string]{
		DB: db, Plan: plan,
		KeyArgs: func(keys []compositeKey) []any {
			tenants := make([]string, len(keys))
			ids := make([]int64, len(keys))
			for i, key := range keys {
				tenants[i], ids[i] = key.Tenant, key.ID
			}
			return []any{tenants, ids}
		},
		Decode: func(rows *sql.Rows) (int64, string, error) {
			var ordinal int64
			var name string
			err := rows.Scan(&ordinal, &name)
			return ordinal, name, err
		},
	}
	req := batchweaver.MustNewBatchRequest([]batchweaver.BatchItem[compositeKey]{
		batchweaver.NewBatchItem(1, compositeKey{Tenant: "acme", ID: 7}),
		batchweaver.NewBatchItem(2, compositeKey{Tenant: "acme", ID: 9}),
	})
	fakeErr = nil
	fakeData = [][]driver.Value{{int64(1), "Ada"}}
	resp, err := provider.Execute(context.Background(), req)
	if err != nil {
		t.Fatal(err)
	}
	if len(fakeArgs) != 2 || !resp.Outcomes()[0].IsSuccess() || !errors.Is(resp.Outcomes()[1].Err, sql.ErrNoRows) {
		t.Fatalf("args=%+v outcomes=%+v", fakeArgs, resp.Outcomes())
	}

	fakeData = [][]driver.Value{{int64(1), "Ada"}, {int64(1), "duplicate"}}
	if _, err := provider.Execute(context.Background(), req); err == nil || !strings.Contains(err.Error(), "duplicate ordinal") {
		t.Fatalf("duplicate join row was not rejected: %v", err)
	}
	provider.Limits = ParameterLimits{MaxItems: 10, MaxParameters: 10, MaxPayloadKB: 1}
	provider.EstimatePayload = func([]compositeKey) int { return 2048 }
	if _, err := provider.Execute(context.Background(), req); err == nil || !strings.Contains(err.Error(), CodeParamLimitExceeded) {
		t.Fatalf("payload limit error = %v", err)
	}
	provider.EstimatePayload = nil
	provider.KeyArgs = func([]compositeKey) []any { return []any{[]string{"acme"}} }
	if _, err := provider.Execute(context.Background(), req); !errors.Is(err, ErrProviderMisconfigured) {
		t.Fatalf("argument mismatch error = %v", err)
	}
}

func TestVerifyReadOnly(t *testing.T) {
	t.Parallel()
	data := map[int]string{1: "a", 2: "b"}
	scalar := func(_ context.Context, k int) (string, error) {
		v, ok := data[k]
		if !ok {
			return "", sql.ErrNoRows
		}
		return v, nil
	}
	batch := func(_ context.Context, ks []int) ([]batchweaver.Outcome[string], error) {
		out := make([]batchweaver.Outcome[string], len(ks))
		for i, k := range ks {
			if v, ok := data[k]; ok {
				out[i] = batchweaver.Success(batchweaver.RequestID(i+1), v)
			} else {
				out[i] = batchweaver.Failure[string](batchweaver.RequestID(i+1), sql.ErrNoRows)
			}
		}
		return out, nil
	}
	vc := VerifyReadOnly(context.Background(), "users.get", "database/sql", scalar, batch,
		func(a, b string) bool { return a == b },
		[]VerifyCase[int]{
			{Name: "unique", Keys: []int{1, 2}},
			{Name: "missing", Keys: []int{1, 9}},
			{Name: "duplicate", Keys: []int{1, 1}},
			{Name: "empty", Keys: nil},
		})
	if !vc.Passed {
		t.Fatalf("verification failed: %+v", vc.Cases)
	}
	if vc.Digest == "" {
		t.Error("missing digest")
	}
}

func TestVerifyReadOnlyCompositeDifferential(t *testing.T) {
	t.Parallel()
	type key struct{ Tenant, ID string }
	data := map[key]string{{Tenant: "acme", ID: "7"}: "Ada", {Tenant: "acme", ID: "9"}: "Lin"}
	scalar := func(_ context.Context, k key) (string, error) {
		value, ok := data[k]
		if !ok {
			return "", sql.ErrNoRows
		}
		return value, nil
	}
	batch := func(_ context.Context, keys []key) ([]batchweaver.Outcome[string], error) {
		outcomes := make([]batchweaver.Outcome[string], len(keys))
		for i, k := range keys {
			if value, ok := data[k]; ok {
				outcomes[i] = batchweaver.Success(batchweaver.RequestID(i+1), value)
			} else {
				outcomes[i] = batchweaver.Failure[string](batchweaver.RequestID(i+1), sql.ErrNoRows)
			}
		}
		return outcomes, nil
	}
	report := VerifyReadOnly(context.Background(), "users.get", "database/sql-composite", scalar, batch,
		func(left, right string) bool { return left == right },
		[]VerifyCase[key]{
			{Name: "unique", Keys: []key{{"acme", "7"}, {"acme", "9"}}},
			{Name: "duplicates", Keys: []key{{"acme", "7"}, {"acme", "7"}}},
			{Name: "missing", Keys: []key{{"acme", "missing"}}},
		})
	if !report.Passed {
		t.Fatalf("composite differential failed: %+v", report.Cases)
	}
}

func planFor(t *testing.T, sqlText, keyType string) SynthPlan {
	t.Helper()
	q, rej := ParseExactKeySelect(sqlText)
	if rej != nil {
		t.Fatal(rej)
	}
	plan, srej := SynthesizeExactKey(SynthInput{Query: q, KeyType: keyType})
	if srej != nil {
		t.Fatal(srej)
	}
	return plan
}
