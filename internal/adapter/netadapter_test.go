package adapter

import (
	"context"
	"encoding/json"
	"io"
	"net/http"
	"net/http/httptest"
	"strconv"
	"strings"
	"testing"

	batchweaver "github.com/Voskan/BatchWeaver"
)

func TestNetworkManifests(t *testing.T) {
	t.Parallel()
	want := map[string]bool{"http/openapi": false, "graphql/gqlgen": false, "grpc-go": false}
	for _, m := range Manifests() {
		if m.Category == CategoryNetwork {
			if _, ok := want[m.AdapterID]; ok {
				want[m.AdapterID] = true
			}
			if err := m.Validate(); err != nil {
				t.Errorf("%s invalid: %v", m.AdapterID, err)
			}
		}
	}
	for id, found := range want {
		if !found {
			t.Errorf("missing network manifest %q", id)
		}
	}
}

func TestParseGraphQLWaves(t *testing.T) {
	t.Parallel()
	src := `query OrderPage($n: Int) {
      orders(first: $n) {
        id
        customer { name }
        product @include(if: true) { title }
        ...orderMeta
      }
    }
    fragment orderMeta on Order { assignee { name } }`
	doc, rej := ParseGraphQL(src)
	if rej != nil {
		t.Fatalf("parse: %v", rej)
	}
	if len(doc.Operations) != 1 || doc.Operations[0].Name != "OrderPage" {
		t.Fatalf("operations: %+v", doc.Operations)
	}
	waves := ResolverWaves(doc, doc.Operations[0])
	if len(waves) < 2 {
		t.Fatalf("expected >=2 waves, got %d: %+v", len(waves), waves)
	}
	if len(waves[0].Fields) != 1 || waves[0].Fields[0] != "orders" {
		t.Errorf("wave 0 = %+v, want [orders]", waves[0].Fields)
	}
	// Wave 1 should include customer, product, and the fragment-expanded assignee.
	joined := strings.Join(waves[1].Fields, ",")
	for _, want := range []string{"orders.customer", "orders.product", "orders.assignee"} {
		if !strings.Contains(joined, want) {
			t.Errorf("wave 1 missing %q: %v", want, waves[1].Fields)
		}
	}
}

func TestParseGraphQLReject(t *testing.T) {
	t.Parallel()
	_, rej := ParseGraphQL("query { orders {")
	if rej == nil || rej.Code != CodeGraphQLParse {
		t.Fatalf("expected parse rejection, got %v", rej)
	}
	// Must not panic on arbitrary input.
	for _, s := range []string{"", "{", "}}}", "query", "...", "fragment", "@@@"} {
		_, _ = ParseGraphQL(s)
	}
}

func TestGraphQLSelectionDigest(t *testing.T) {
	t.Parallel()
	doc, _ := ParseGraphQL(`query { a { x { id } } b { x { id name } } }`)
	op := doc.Operations[0]
	var ax, bx GraphQLField
	for _, f := range op.Selection.Fields {
		if f.Name == "a" {
			ax = f.Sel.Fields[0]
		}
		if f.Name == "b" {
			bx = f.Sel.Fields[0]
		}
	}
	if NormalizeSelectionDigest(doc, ax) == NormalizeSelectionDigest(doc, bx) {
		t.Error("different sub-selections should yield different digests")
	}
}

func TestGRPCBindingAndMetadata(t *testing.T) {
	t.Parallel()
	if rej := (GRPCBinding{}).Validate(); rej == nil {
		t.Error("empty binding should be rejected")
	}
	ok := GRPCBinding{ScalarMethod: "/s/Get", BatchMethod: "/s/Batch", RequestKey: "id", BatchRequestsField: "requests", ResponseMode: GRPCKeyed, ResponseKey: "id"}
	if rej := ok.Validate(); rej != nil {
		t.Errorf("valid binding rejected: %v", rej)
	}
	if ClassifyMetadata("authorization") != MetaPartition {
		t.Error("authorization must partition")
	}
	if ClassifyMetadata("x-tenant-id") != MetaPartition {
		t.Error("x-tenant-id must partition")
	}
	if ClassifyMetadata("traceparent") != MetaMerge {
		t.Error("traceparent should merge")
	}
	if ClassifyMetadata("content-type") != MetaMustEqual {
		t.Error("content-type must-equal")
	}
}

func TestLoadOpenAPI(t *testing.T) {
	t.Parallel()
	yamlDoc := `openapi: 3.1.0
paths:
  /users:batchGet:
    post:
      operationId: batchGetUsers
      x-batchweaver:
        scalar-operation-id: users.get
        mode: keyed
        request-items-path: /items
        request-key-path: /key
        response-items-path: /items
        response-key-path: /request_id
        maximum-items: 500
`
	bindings, rej := LoadOpenAPI([]byte(yamlDoc), true)
	if rej != nil {
		t.Fatalf("load: %v", rej)
	}
	if len(bindings) != 1 || bindings[0].ScalarOperationID != "users.get" {
		t.Fatalf("bindings: %+v", bindings)
	}
	if bindings[0].Binding.Mode != EnvelopeKeyed || bindings[0].Binding.MaxItems != 500 {
		t.Errorf("binding: %+v", bindings[0].Binding)
	}

	// Unsupported version rejected.
	if _, rej := LoadOpenAPI([]byte(`openapi: 2.0`), true); rej == nil {
		t.Error("expected rejection for OpenAPI 2.0")
	}
	// Keyed extension without response-key rejected.
	bad := `openapi: 3.1.0
paths:
  /x:
    post:
      x-batchweaver:
        scalar-operation-id: x.get
        mode: keyed
`
	if _, rej := LoadOpenAPI([]byte(bad), true); rej == nil || rej.Code != CodeHTTPCorrelationMissing {
		t.Errorf("expected correlation-missing rejection, got %v", rej)
	}
}

func TestHTTPProviderKeyed(t *testing.T) {
	t.Parallel()
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		var env struct {
			Items []struct {
				RequestID string `json:"request_id"`
				Key       int    `json:"key"`
			} `json:"items"`
		}
		body, _ := io.ReadAll(r.Body)
		_ = json.Unmarshal(body, &env)
		type respItem struct {
			RequestID string  `json:"request_id"`
			Value     *string `json:"value"`
			Error     string  `json:"error"`
		}
		var out struct {
			Items []respItem `json:"items"`
		}
		for _, it := range env.Items {
			switch it.Key {
			case 99: // missing: omit from response
				continue
			case 13:
				out.Items = append(out.Items, respItem{RequestID: it.RequestID, Error: "boom"})
			default:
				v := "v" + strconv.Itoa(it.Key)
				out.Items = append(out.Items, respItem{RequestID: it.RequestID, Value: &v})
			}
		}
		w.Header().Set("Content-Type", "application/json")
		_ = json.NewEncoder(w).Encode(out)
	}))
	defer srv.Close()

	prov := HTTPProvider[int, string]{
		Client:  srv.Client(),
		Binding: HTTPBinding{Method: http.MethodPost, URL: srv.URL, Mode: EnvelopeKeyed, MaxItems: 100},
	}
	req := batchweaver.MustNewBatchRequest([]batchweaver.BatchItem[int]{
		batchweaver.NewBatchItem(1, 5), batchweaver.NewBatchItem(2, 99),
		batchweaver.NewBatchItem(3, 13), batchweaver.NewBatchItem(4, 5),
	})
	resp, err := prov.Execute(context.Background(), req)
	if err != nil {
		t.Fatalf("execute: %v", err)
	}
	outs := resp.Outcomes()
	if !outs[0].IsSuccess() || outs[0].Value != "v5" {
		t.Errorf("item 0: %+v", outs[0])
	}
	if outs[1].Err == nil {
		t.Errorf("item 1 (key 99) should be missing")
	}
	if outs[2].Err == nil || outs[2].Err.Error() != "boom" {
		t.Errorf("item 2 error: %+v", outs[2])
	}
	if !outs[3].IsSuccess() || outs[3].Value != "v5" { // duplicate key, distinct ordinal
		t.Errorf("item 3 (duplicate): %+v", outs[3])
	}
}

func TestHTTPProviderPositionalLengthMismatch(t *testing.T) {
	t.Parallel()
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, _ *http.Request) {
		_, _ = io.WriteString(w, `{"items":[]}`) // zero items for a non-empty request
	}))
	defer srv.Close()
	prov := HTTPProvider[int, string]{Client: srv.Client(), Binding: HTTPBinding{URL: srv.URL, Mode: EnvelopePositional}}
	req := batchweaver.MustNewBatchRequest([]batchweaver.BatchItem[int]{batchweaver.NewBatchItem(1, 1)})
	if _, err := prov.Execute(context.Background(), req); err == nil {
		t.Fatal("expected positional length mismatch error")
	}
}
