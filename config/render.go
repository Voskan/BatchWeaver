package config

import (
	"bytes"
	"encoding/json"
	"sort"

	"github.com/Voskan/BatchWeaver/internal/configdecode"
	"github.com/Voskan/BatchWeaver/operation"
)

// RenderJSON renders the normalized configuration as deterministic, canonical
// JSON: stable field order, operations sorted by ID, durations and byte sizes as
// canonical strings, no source positions or include directives, and a trailing
// newline. The same semantic configuration always produces identical bytes.
func RenderJSON(cfg Config) ([]byte, error) {
	doc := configJSON{
		Version:       cfg.Version,
		Compiler:      compilerJSON{Mode: cfg.Compiler.Mode.String()},
		Runtime:       runtimeJSON{DefaultScope: cfg.Runtime.DefaultScope.String()},
		Security:      securityJSON{CrossScopeBatching: cfg.Security.CrossScopeBatching, RawKeyObservability: cfg.Security.RawKeyObservability},
		Observability: observabilityJSON{Metrics: cfg.Observability.Metrics, Tracing: cfg.Observability.Tracing, Logging: cfg.Observability.Logging.String()},
	}
	for _, sp := range cfg.Catalog.List() {
		doc.Operations = append(doc.Operations, operationToJSON(sp))
	}
	exts := append([]operation.Extension(nil), cfg.Extensions...)
	sort.Slice(exts, func(i, j int) bool { return exts[i].Namespace < exts[j].Namespace })
	for _, e := range exts {
		doc.Extensions = append(doc.Extensions, extensionJSON{Namespace: e.Namespace, Data: json.RawMessage(e.Data)})
	}

	var buf bytes.Buffer
	enc := json.NewEncoder(&buf)
	enc.SetEscapeHTML(false)
	enc.SetIndent("", "  ")
	if err := enc.Encode(doc); err != nil {
		return nil, err
	}
	return buf.Bytes(), nil
}

type configJSON struct {
	Version       int               `json:"version"`
	Compiler      compilerJSON      `json:"compiler"`
	Runtime       runtimeJSON       `json:"runtime"`
	Security      securityJSON      `json:"security"`
	Observability observabilityJSON `json:"observability"`
	Operations    []operationJSON   `json:"operations"`
	Extensions    []extensionJSON   `json:"extensions,omitempty"`
}

type compilerJSON struct {
	Mode string `json:"mode"`
}

type runtimeJSON struct {
	DefaultScope string `json:"default_scope"`
}

type securityJSON struct {
	CrossScopeBatching  bool `json:"cross_scope_batching"`
	RawKeyObservability bool `json:"raw_key_observability"`
}

type observabilityJSON struct {
	Metrics bool   `json:"metrics"`
	Tracing bool   `json:"tracing"`
	Logging string `json:"logging"`
}

type extensionJSON struct {
	Namespace string          `json:"namespace"`
	Data      json.RawMessage `json:"data"`
}

type operationJSON struct {
	ID            string            `json:"id"`
	Kind          string            `json:"kind"`
	Scalar        string            `json:"scalar,omitempty"`
	Batch         string            `json:"batch,omitempty"`
	Results       resultsJSON       `json:"results"`
	Partition     partitionJSON     `json:"partition"`
	Scheduler     schedulerJSON     `json:"scheduler"`
	Deduplication deduplicationJSON `json:"deduplication"`
	Retry         retryJSON         `json:"retry"`
	Fallback      fallbackJSON      `json:"fallback"`
}

type resultsJSON struct {
	Mode    string `json:"mode"`
	Missing string `json:"missing"`
	Errors  string `json:"errors"`
}

type partitionJSON struct {
	Scope      string   `json:"scope"`
	Dimensions []string `json:"dimensions,omitempty"`
}

type schedulerJSON struct {
	Mode           string `json:"mode"`
	MinSize        int    `json:"min_size"`
	MaxSize        int    `json:"max_size"`
	MaxWeight      int    `json:"max_weight"`
	MaxPayload     string `json:"max_payload"`
	MaxWait        string `json:"max_wait"`
	DeadlineMargin string `json:"deadline_margin"`
	MaxConcurrency int    `json:"max_concurrency"`
	QueueItems     int    `json:"queue_items"`
	QueueBytes     string `json:"queue_bytes"`
}

type deduplicationJSON struct {
	Mode     string `json:"mode"`
	InFlight bool   `json:"inflight"`
	MaxItems int    `json:"max_items,omitempty"`
}

type retryJSON struct {
	Enabled         bool     `json:"enabled"`
	MaximumAttempts int      `json:"maximum_attempts,omitempty"`
	InitialBackoff  string   `json:"initial_backoff,omitempty"`
	MaximumBackoff  string   `json:"maximum_backoff,omitempty"`
	Retryable       []string `json:"retryable,omitempty"`
}

type fallbackJSON struct {
	Mode string `json:"mode"`
}

func operationToJSON(sp operation.Spec) operationJSON {
	out := operationJSON{
		ID:   sp.ID().String(),
		Kind: sp.Semantics().Kind().String(),
	}
	if !sp.ScalarSymbol().IsZero() {
		out.Scalar = sp.ScalarSymbol().String()
	}
	if !sp.BatchSymbol().IsZero() {
		out.Batch = sp.BatchSymbol().String()
	}
	r := sp.ResultContract()
	out.Results = resultsJSON{Mode: r.Mode().String(), Missing: r.Missing().String(), Errors: r.ErrorMode().String()}
	p := sp.PartitionContract()
	dims := p.Required()
	sort.Slice(dims, func(i, j int) bool { return dims[i] < dims[j] })
	out.Partition.Scope = p.Scope().String()
	for _, d := range dims {
		out.Partition.Dimensions = append(out.Partition.Dimensions, d.String())
	}
	s := sp.SchedulerPolicy()
	out.Scheduler = schedulerJSON{
		Mode: s.Mode().String(), MinSize: s.MinBatchSize(), MaxSize: s.MaxBatchSize(),
		MaxWeight: s.MaxBatchWeight(), MaxPayload: ByteSize(s.MaxPayloadBytes()).String(),
		MaxWait: Duration(s.MaxWait()).String(), DeadlineMargin: Duration(s.DeadlineMargin()).String(),
		MaxConcurrency: s.MaxConcurrency(), QueueItems: s.QueueItems(), QueueBytes: ByteSize(s.QueueBytes()).String(),
	}
	d := sp.DeduplicationPolicy()
	out.Deduplication = deduplicationJSON{Mode: d.Mode().String(), InFlight: d.InFlight(), MaxItems: d.MaxItems()}
	rt := sp.RetryPolicy()
	out.Retry = retryJSON{Enabled: rt.Enabled()}
	if rt.Enabled() {
		out.Retry.MaximumAttempts = rt.MaxAttempts()
		out.Retry.InitialBackoff = Duration(rt.InitialBackoff()).String()
		out.Retry.MaximumBackoff = Duration(rt.MaxBackoff()).String()
		for _, c := range rt.Retryable() {
			out.Retry.Retryable = append(out.Retry.Retryable, c.String())
		}
	}
	out.Fallback = fallbackJSON{Mode: sp.FallbackPolicy().Mode().String()}
	return out
}

// encodeNodeJSON renders a decoded node subtree as compact, deterministic JSON.
// It is used to preserve uninterpreted extension data.
func encodeNodeJSON(node *configdecode.Node) []byte {
	var buf bytes.Buffer
	writeNodeJSON(&buf, node)
	return buf.Bytes()
}

func writeNodeJSON(buf *bytes.Buffer, node *configdecode.Node) {
	switch {
	case node == nil || node.IsNull():
		buf.WriteString("null")
	case node.IsMapping():
		buf.WriteByte('{')
		for i, e := range node.Entries {
			if i > 0 {
				buf.WriteByte(',')
			}
			writeJSONString(buf, e.Key)
			buf.WriteByte(':')
			writeNodeJSON(buf, e.Value)
		}
		buf.WriteByte('}')
	case node.IsSequence():
		buf.WriteByte('[')
		for i, e := range node.Elems {
			if i > 0 {
				buf.WriteByte(',')
			}
			writeNodeJSON(buf, e)
		}
		buf.WriteByte(']')
	default:
		switch node.ScalarType {
		case configdecode.ScalarInt, configdecode.ScalarFloat:
			buf.WriteString(node.Value)
		case configdecode.ScalarBool:
			buf.WriteString(node.Value)
		default:
			writeJSONString(buf, node.Value)
		}
	}
}

func writeJSONString(buf *bytes.Buffer, s string) {
	b, _ := json.Marshal(s)
	buf.Write(b)
}
