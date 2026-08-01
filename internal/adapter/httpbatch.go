package adapter

import (
	"bytes"
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"io"
	"net/http"
	"strconv"

	batchweaver "github.com/Voskan/BatchWeaver"
)

// EnvelopeMode selects how a batch response correlates to the request.
type EnvelopeMode string

const (
	// EnvelopeKeyed correlates response items by their request_id field.
	EnvelopeKeyed EnvelopeMode = "keyed"
	// EnvelopePositional correlates response items by position; the response must
	// contain exactly one item per request item in order.
	EnvelopePositional EnvelopeMode = "positional"
)

// HTTPBinding declares an explicit HTTP batch endpoint. It is never inferred from
// endpoint naming; it must be declared through configuration or an OpenAPI vendor
// extension.
type HTTPBinding struct {
	Method   string
	URL      string
	Mode     EnvelopeMode
	MaxItems int
}

// ErrHTTPMissingItem is the default outcome error for a requested key absent from
// a keyed batch response.
var ErrHTTPMissingItem = errors.New("adapter: batch response is missing the requested item")

// httpReqItem is one request item in the typed JSON envelope.
type httpReqItem[K any] struct {
	RequestID string `json:"request_id"`
	Key       K      `json:"key"`
}

type httpReqEnvelope[K any] struct {
	Items []httpReqItem[K] `json:"items"`
}

// httpRespItem is one response item in the typed JSON envelope. Value is a pointer
// so absent, null, and zero value remain distinguishable.
type httpRespItem[V any] struct {
	RequestID string `json:"request_id"`
	Value     *V     `json:"value"`
	Error     string `json:"error"`
}

type httpRespEnvelope[V any] struct {
	Items []httpRespItem[V] `json:"items"`
}

// HTTPProvider executes an explicit HTTP batch endpoint with a typed JSON
// envelope and maps the response back to ordered scalar outcomes. It preserves
// order and duplicates, validates correlation, and chunks deterministically. It
// uses the caller-owned *http.Client, preserving its transport, TLS, cookie jar,
// redirect policy, and auth identity.
type HTTPProvider[K any, V any] struct {
	Client  *http.Client
	Binding HTTPBinding
	// Missing returns the outcome error for a requested key absent from a keyed
	// response; it defaults to ErrHTTPMissingItem.
	Missing func() error
}

// Execute issues the batch request(s) and returns one outcome per request item.
func (p HTTPProvider[K, V]) Execute(ctx context.Context, req batchweaver.BatchRequest[K]) (batchweaver.BatchResponse[V], error) {
	client := p.Client
	if client == nil {
		client = http.DefaultClient
	}
	missing := p.Missing
	if missing == nil {
		missing = func() error { return ErrHTTPMissingItem }
	}
	items := req.Items()
	limits := ParameterLimits{MaxItems: p.Binding.MaxItems}
	if limits.MaxItems <= 0 {
		limits.MaxItems = 500
	}

	outcomes := make([]batchweaver.Outcome[V], 0, len(items))
	for _, rng := range Chunks(len(items), limits) {
		chunk := items[rng[0]:rng[1]]
		chunkOut, err := p.executeChunk(ctx, client, chunk, missing)
		if err != nil {
			return batchweaver.BatchResponse[V]{}, err
		}
		outcomes = append(outcomes, chunkOut...)
	}
	return batchweaver.NewBatchResponse(outcomes)
}

func (p HTTPProvider[K, V]) executeChunk(ctx context.Context, client *http.Client, chunk []batchweaver.BatchItem[K], missing func() error) ([]batchweaver.Outcome[V], error) {
	env := httpReqEnvelope[K]{Items: make([]httpReqItem[K], len(chunk))}
	for i, it := range chunk {
		env.Items[i] = httpReqItem[K]{RequestID: strconv.FormatUint(uint64(it.ID), 10), Key: it.Key}
	}
	body, err := json.Marshal(env)
	if err != nil {
		return nil, err
	}
	method := p.Binding.Method
	if method == "" {
		method = http.MethodPost
	}
	httpReq, err := http.NewRequestWithContext(ctx, method, p.Binding.URL, bytes.NewReader(body))
	if err != nil {
		return nil, err
	}
	httpReq.Header.Set("Content-Type", "application/json")
	resp, err := client.Do(httpReq)
	if err != nil {
		return nil, err
	}
	defer func() { _ = resp.Body.Close() }()
	respBody, err := io.ReadAll(resp.Body)
	if err != nil {
		return nil, err
	}
	if resp.StatusCode < 200 || resp.StatusCode >= 300 {
		return nil, fmt.Errorf("%s: batch endpoint returned status %d", CodeVerificationFailed, resp.StatusCode)
	}
	var decoded httpRespEnvelope[V]
	if err := json.Unmarshal(respBody, &decoded); err != nil {
		return nil, fmt.Errorf("decode batch response: %w", err)
	}
	if p.Binding.Mode == EnvelopePositional {
		return p.mapPositional(chunk, decoded, missing)
	}
	return p.mapKeyed(chunk, decoded, missing)
}

func (p HTTPProvider[K, V]) mapKeyed(chunk []batchweaver.BatchItem[K], decoded httpRespEnvelope[V], missing func() error) ([]batchweaver.Outcome[V], error) {
	byID := make(map[string]httpRespItem[V], len(decoded.Items))
	for _, it := range decoded.Items {
		if _, dup := byID[it.RequestID]; dup {
			return nil, fmt.Errorf("%s: duplicate response request_id %q", CodeVerificationFailed, it.RequestID)
		}
		byID[it.RequestID] = it
	}
	out := make([]batchweaver.Outcome[V], len(chunk))
	for i, it := range chunk {
		id := strconv.FormatUint(uint64(it.ID), 10)
		r, ok := byID[id]
		switch {
		case !ok:
			out[i] = batchweaver.Failure[V](it.ID, missing())
		case r.Error != "":
			out[i] = batchweaver.Failure[V](it.ID, errors.New(r.Error))
		case r.Value == nil:
			out[i] = batchweaver.Failure[V](it.ID, missing())
		default:
			out[i] = batchweaver.Success(it.ID, *r.Value)
		}
	}
	return out, nil
}

func (p HTTPProvider[K, V]) mapPositional(chunk []batchweaver.BatchItem[K], decoded httpRespEnvelope[V], missing func() error) ([]batchweaver.Outcome[V], error) {
	if len(decoded.Items) != len(chunk) {
		return nil, fmt.Errorf("%s: positional response has %d items for %d requests", CodeVerificationFailed, len(decoded.Items), len(chunk))
	}
	out := make([]batchweaver.Outcome[V], len(chunk))
	for i, it := range chunk {
		r := decoded.Items[i]
		switch {
		case r.Error != "":
			out[i] = batchweaver.Failure[V](it.ID, errors.New(r.Error))
		case r.Value == nil:
			out[i] = batchweaver.Failure[V](it.ID, missing())
		default:
			out[i] = batchweaver.Success(it.ID, *r.Value)
		}
	}
	return out, nil
}
