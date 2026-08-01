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
	if _, ok := ManifestByID("database/sql"); !ok {
		t.Error("database/sql manifest missing")
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
		{"SELECT id FROM a JOIN b ON a.x = b.x WHERE a.id = $1", CodeSQLUnsupported},
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
func (c *fakeConn) QueryContext(ctx context.Context, _ string, _ []driver.NamedValue) (driver.Rows, error) {
	if err := ctx.Err(); err != nil {
		return nil, err
	}
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
