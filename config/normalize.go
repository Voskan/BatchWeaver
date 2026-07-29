package config

import (
	"errors"
	"fmt"
	"sort"
	"time"

	"github.com/Voskan/BatchWeaver/diagnostics"
	"github.com/Voskan/BatchWeaver/internal/configdecode"
	"github.com/Voskan/BatchWeaver/operation"
)

// Known field names at each level, used for strict unknown-field detection.
var (
	topLevelFields  = []string{"version", "include", "compiler", "runtime", "security", "observability", "operations", "extensions"}
	compilerFields  = []string{"mode"}
	runtimeFields   = []string{"default_scope"}
	securityFields  = []string{"cross_scope_batching", "raw_key_observability"}
	observeFields   = []string{"metrics", "tracing", "logging"}
	operationFields = []string{"scalar", "batch", "kind", "results", "partition", "scheduler", "deduplication", "retry", "fallback", "replace", "extensions"}
	symbolFields    = []string{"symbol"}
	resultsFields   = []string{"mode", "missing", "errors"}
	partitionFields = []string{"scope", "dimensions"}
	schedulerFields = []string{"mode", "min_size", "max_size", "max_weight", "max_payload", "max_wait", "deadline_margin", "max_concurrency", "queue_items", "queue_bytes", "active_partitions", "waiters_per_key"}
	dedupFields     = []string{"mode", "inflight", "scope_memoization", "negative_memoization", "error_memoization", "max_items", "max_bytes", "canonicalizer"}
	retryFields     = []string{"enabled", "maximum_attempts", "initial_backoff", "maximum_backoff", "jitter", "retryable", "respect_retry_after", "partial_item_retry", "unknown_outcome", "idempotency_key"}
	fallbackFields  = []string{"mode", "max_scalar_concurrency", "on_queue_overflow", "on_provider_unavailable", "on_unsupported_partition"}
)

// normalizer walks a merged node tree and builds a typed Config and catalog,
// recording diagnostics with source positions.
type normalizer struct {
	diags diagnostics.Collection
}

// normalize converts a merged configuration node into a Config and operation
// Catalog. It always returns a Config (with defaults) and the diagnostics it
// found; callers must check for errors in the diagnostics.
func normalize(node *configdecode.Node) (Config, operation.Catalog, diagnostics.Collection) {
	n := &normalizer{}
	cfg := defaultConfig()

	if node == nil || !node.IsMapping() {
		n.add(configdecode.CodeTypeMismatch, positionOf(node), "configuration must be a mapping")
		return cfg, operation.Catalog{}, n.diags
	}
	configdecode.CheckUnknownFields(node, topLevelFields, &n.diags)

	n.version(node, &cfg)
	n.compiler(node, &cfg)
	n.runtime(node, &cfg)
	n.security(node, &cfg)
	n.observability(node, &cfg)
	cfg.Extensions = n.extensions(node, "extensions")
	catalog := n.operations(node)
	cfg.Catalog = catalog
	return cfg, catalog, n.diags
}

func (n *normalizer) add(code diagnostics.Code, pos diagnostics.Position, msg string) {
	n.diags.Add(diagnostics.Diagnostic{
		Code: code, Severity: diagnostics.SeverityError, Message: msg,
		Source: "config", Range: diagnostics.AtPosition(pos),
	})
}

func (n *normalizer) version(node *configdecode.Node, cfg *Config) {
	v, pos, ok := node.Get("version")
	if !ok {
		n.add(configdecode.CodeMissingField, node.Pos, "missing required field \"version\"")
		return
	}
	iv, ok := configdecode.AsInt(v)
	if !ok {
		n.add(configdecode.CodeTypeMismatch, pos, "field \"version\" must be an integer")
		return
	}
	if !SchemaVersionSupported(int(iv)) {
		n.add(configdecode.CodeUnsupportedVersion, v.Pos, fmt.Sprintf("unsupported schema version %d", iv))
		return
	}
	cfg.Version = int(iv)
}

func (n *normalizer) compiler(node *configdecode.Node, cfg *Config) {
	m, ok := n.mapping(node, "compiler")
	if !ok {
		return
	}
	configdecode.CheckUnknownFields(m, compilerFields, &n.diags)
	if s, vn, ok := n.str(m, "mode"); ok {
		mode, err := ParseCompilerMode(s)
		if err != nil {
			n.add(configdecode.CodeInvalidEnum, vn.Pos, fmt.Sprintf("invalid compiler mode %q", s))
		} else {
			cfg.Compiler.Mode = mode
		}
	}
}

func (n *normalizer) runtime(node *configdecode.Node, cfg *Config) {
	m, ok := n.mapping(node, "runtime")
	if !ok {
		return
	}
	configdecode.CheckUnknownFields(m, runtimeFields, &n.diags)
	if s, vn, ok := n.str(m, "default_scope"); ok {
		scope, err := operation.ParseScope(s)
		if err != nil {
			n.add(configdecode.CodeInvalidEnum, vn.Pos, fmt.Sprintf("invalid default scope %q", s))
		} else {
			cfg.Runtime.DefaultScope = scope
		}
	}
}

func (n *normalizer) security(node *configdecode.Node, cfg *Config) {
	m, ok := n.mapping(node, "security")
	if !ok {
		return
	}
	configdecode.CheckUnknownFields(m, securityFields, &n.diags)
	if b, ok := n.boolField(m, "cross_scope_batching"); ok {
		cfg.Security.CrossScopeBatching = b
	}
	if b, ok := n.boolField(m, "raw_key_observability"); ok {
		cfg.Security.RawKeyObservability = b
	}
}

func (n *normalizer) observability(node *configdecode.Node, cfg *Config) {
	m, ok := n.mapping(node, "observability")
	if !ok {
		return
	}
	configdecode.CheckUnknownFields(m, observeFields, &n.diags)
	if b, ok := n.boolField(m, "metrics"); ok {
		cfg.Observability.Metrics = b
	}
	if b, ok := n.boolField(m, "tracing"); ok {
		cfg.Observability.Tracing = b
	}
	if s, vn, ok := n.str(m, "logging"); ok {
		lvl, err := ParseLoggingLevel(s)
		if err != nil {
			n.add(configdecode.CodeInvalidEnum, vn.Pos, fmt.Sprintf("invalid logging level %q", s))
		} else {
			cfg.Observability.Logging = lvl
		}
	}
}

// operations builds the operation catalog from the operations mapping.
func (n *normalizer) operations(node *configdecode.Node) operation.Catalog {
	m, ok := n.mapping(node, "operations")
	if !ok {
		return operation.Catalog{}
	}
	builder := operation.NewCatalogBuilder()
	for _, entry := range m.Entries {
		spec, ok := n.operation(entry.Key, entry.KeyPos, entry.Value)
		if ok {
			builder.Add(spec)
		}
	}
	n.diags.AddCollection(builder.Diagnostics())
	return builder.Build()
}

// operation builds a single operation.Spec from a node.
func (n *normalizer) operation(idText string, idPos diagnostics.Position, node *configdecode.Node) (operation.Spec, bool) {
	id, err := operation.ParseID(idText)
	if err != nil {
		n.add(configdecode.CodeInvalidValue, idPos, fmt.Sprintf("invalid operation id %q: %v", idText, err))
		return operation.Spec{}, false
	}
	if !node.IsMapping() {
		n.add(configdecode.CodeTypeMismatch, positionOf(node), fmt.Sprintf("operation %q must be a mapping", idText))
		return operation.Spec{}, false
	}
	configdecode.CheckUnknownFields(node, operationFields, &n.diags)

	sem, ok := n.semantics(node)
	if !ok {
		return operation.Spec{}, false
	}

	opts := []operation.SpecOption{
		operation.WithSource(operation.Source{Range: diagnostics.AtPosition(idPos)}),
	}
	if sym, ok := n.symbol(node, "scalar"); ok {
		opts = append(opts, operation.WithScalarSymbol(sym))
	}
	if sym, ok := n.symbol(node, "batch"); ok {
		opts = append(opts, operation.WithBatchSymbol(sym))
	}
	opts = append(opts, operation.WithResultContract(n.resultContract(node)))
	opts = append(opts, operation.WithPartitionContract(n.partitionContract(node)))
	opts = append(opts, operation.WithSchedulerPolicy(n.schedulerPolicy(node)))
	opts = append(opts, operation.WithDeduplicationPolicy(n.deduplicationPolicy(node)))
	opts = append(opts, operation.WithRetryPolicy(n.retryPolicy(node)))
	opts = append(opts, operation.WithFallbackPolicy(n.fallbackPolicy(node)))
	if exts := n.extensions(node, "extensions"); len(exts) > 0 {
		opts = append(opts, operation.WithExtensions(exts))
	}

	// Build without failing on validation here; the catalog builder validates and
	// records diagnostics, so a spec with issues still produces useful findings.
	spec, err := operation.NewSpec(id, sem, opts...)
	if err != nil {
		var verr *operation.ValidationError
		if errors.As(err, &verr) {
			n.diags.AddCollection(verr.Diagnostics)
			return operation.Spec{}, false
		}
		n.add(configdecode.CodeSemantic, idPos, err.Error())
		return operation.Spec{}, false
	}
	return spec, true
}

// semantics resolves the kind field into base semantics.
func (n *normalizer) semantics(node *configdecode.Node) (operation.Semantics, bool) {
	s, vn, ok := n.str(node, "kind")
	if !ok {
		n.add(configdecode.CodeMissingField, node.Pos, "operation is missing required field \"kind\"")
		return operation.Semantics{}, false
	}
	kind, err := operation.ParseKind(s)
	if err != nil {
		n.add(configdecode.CodeInvalidEnum, vn.Pos, fmt.Sprintf("invalid operation kind %q", s))
		return operation.Semantics{}, false
	}
	switch kind {
	case operation.KindReadOnly:
		return operation.ReadOnly(), true
	case operation.KindFreshnessSensitiveRead:
		return operation.FreshnessSensitiveRead(), true
	case operation.KindIdempotentWrite:
		return operation.IdempotentWrite(), true
	case operation.KindNonIdempotentWrite:
		return operation.NonIdempotentWrite(), true
	case operation.KindCommutativeAggregation:
		return operation.CommutativeAggregation(), true
	case operation.KindOrderedAggregation:
		return operation.OrderedAggregation(), true
	case operation.KindAtomicGroup:
		return operation.AtomicGroup(), true
	case operation.KindTransactionBound:
		return operation.TransactionBound(), true
	case operation.KindSessionBound:
		return operation.SessionBound(), true
	default:
		n.add(configdecode.CodeInvalidEnum, vn.Pos, fmt.Sprintf("unsupported operation kind %q", s))
		return operation.Semantics{}, false
	}
}

// symbol reads a "<key>.symbol" string into an operation.Symbol.
func (n *normalizer) symbol(node *configdecode.Node, key string) (operation.Symbol, bool) {
	m, ok := n.mapping(node, key)
	if !ok {
		return operation.Symbol{}, false
	}
	configdecode.CheckUnknownFields(m, symbolFields, &n.diags)
	s, vn, ok := n.str(m, "symbol")
	if !ok {
		return operation.Symbol{}, false
	}
	sym, err := operation.ParseSymbol(s)
	if err != nil {
		n.add(configdecode.CodeInvalidValue, vn.Pos, fmt.Sprintf("invalid symbol %q: %v", s, err))
		return operation.Symbol{}, false
	}
	return sym, true
}

func (n *normalizer) resultContract(node *configdecode.Node) operation.ResultContract {
	m, ok := n.mapping(node, "results")
	if !ok {
		return operation.OrderedResults().WithMissing(defaultMissingBehavior)
	}
	configdecode.CheckUnknownFields(m, resultsFields, &n.diags)

	mode := defaultResultMode
	if s, vn, ok := n.str(m, "mode"); ok {
		v, err := operation.ParseResultMode(s)
		if err != nil {
			n.add(configdecode.CodeInvalidEnum, vn.Pos, fmt.Sprintf("invalid result mode %q", s))
		} else {
			mode = v
		}
	}
	missing := defaultMissingBehavior
	missingSet := false
	if s, vn, ok := n.str(m, "missing"); ok {
		v, err := operation.ParseMissingBehavior(s)
		if err != nil {
			n.add(configdecode.CodeInvalidEnum, vn.Pos, fmt.Sprintf("invalid missing behavior %q", s))
		} else {
			missing, missingSet = v, true
		}
	}
	errMode := defaultErrorMode
	if s, vn, ok := n.str(m, "errors"); ok {
		v, err := operation.ParseErrorMode(s)
		if err != nil {
			n.add(configdecode.CodeInvalidEnum, vn.Pos, fmt.Sprintf("invalid error mode %q", s))
		} else {
			errMode = v
		}
	}

	switch mode {
	case operation.ResultSparse:
		if !missingSet || missing == operation.MissingContractViolation {
			n.add(configdecode.CodeInvalidValue, m.Pos, "sparse results require an explicit \"missing\" behavior other than contract-violation")
			missing = operation.MissingError
		}
		return operation.SparseResults(missing).WithErrorMode(errMode)
	case operation.ResultKeyed:
		return operation.KeyedResults().WithMissing(missing).WithErrorMode(errMode)
	default:
		return operation.OrderedResults().WithMissing(missing).WithErrorMode(errMode)
	}
}

func (n *normalizer) partitionContract(node *configdecode.Node) operation.PartitionContract {
	m, ok := n.mapping(node, "partition")
	if !ok {
		return operation.DefaultPartitionContract()
	}
	configdecode.CheckUnknownFields(m, partitionFields, &n.diags)
	scope := defaultScope
	if s, vn, ok := n.str(m, "scope"); ok {
		v, err := operation.ParseScope(s)
		if err != nil {
			n.add(configdecode.CodeInvalidEnum, vn.Pos, fmt.Sprintf("invalid scope %q", s))
		} else {
			scope = v
		}
	}
	var required []operation.PartitionDimension
	if seq, _, ok := m.Get("dimensions"); ok {
		if !seq.IsSequence() {
			n.add(configdecode.CodeTypeMismatch, seq.Pos, "field \"dimensions\" must be a sequence")
		} else {
			for _, e := range seq.Elems {
				s, ok := configdecode.AsString(e)
				if !ok {
					n.add(configdecode.CodeTypeMismatch, e.Pos, "dimension must be a string")
					continue
				}
				dim := operation.PartitionDimension(s)
				if err := dim.Validate(); err != nil {
					n.add(configdecode.CodeInvalidValue, e.Pos, fmt.Sprintf("invalid dimension %q: %v", s, err))
					continue
				}
				required = append(required, dim)
			}
		}
	}
	return operation.NewPartitionContract(scope, required, nil)
}

func (n *normalizer) schedulerPolicy(node *configdecode.Node) operation.SchedulerPolicy {
	def := operation.DefaultSchedulerPolicy()
	m, ok := n.mapping(node, "scheduler")
	if !ok {
		return def
	}
	configdecode.CheckUnknownFields(m, schedulerFields, &n.diags)
	params := operation.SchedulerParams{
		Mode: def.Mode(), MinBatchSize: def.MinBatchSize(), MaxBatchSize: def.MaxBatchSize(),
		MaxBatchWeight: def.MaxBatchWeight(), MaxPayloadBytes: def.MaxPayloadBytes(),
		MaxWait: def.MaxWait(), DeadlineMargin: def.DeadlineMargin(), MaxConcurrency: def.MaxConcurrency(),
		QueueItems: def.QueueItems(), QueueBytes: def.QueueBytes(), ActivePartitions: def.ActivePartitions(),
		WaitersPerKey: def.WaitersPerKey(), Fairness: def.Fairness(), Overflow: def.Overflow(),
	}
	if s, vn, ok := n.str(m, "mode"); ok {
		v, err := operation.ParseSchedulerMode(s)
		if err != nil {
			n.add(configdecode.CodeInvalidEnum, vn.Pos, fmt.Sprintf("invalid scheduler mode %q", s))
		} else {
			params.Mode = v
		}
	}
	params.MinBatchSize = n.intOr(m, "min_size", params.MinBatchSize)
	params.MaxBatchSize = n.intOr(m, "max_size", params.MaxBatchSize)
	params.MaxBatchWeight = n.intOr(m, "max_weight", params.MaxBatchWeight)
	params.MaxConcurrency = n.intOr(m, "max_concurrency", params.MaxConcurrency)
	params.QueueItems = n.intOr(m, "queue_items", params.QueueItems)
	params.ActivePartitions = n.intOr(m, "active_partitions", params.ActivePartitions)
	params.WaitersPerKey = n.intOr(m, "waiters_per_key", params.WaitersPerKey)
	params.MaxPayloadBytes = n.byteSizeOr(m, "max_payload", params.MaxPayloadBytes)
	params.QueueBytes = n.byteSizeOr(m, "queue_bytes", params.QueueBytes)
	params.MaxWait = n.durationOr(m, "max_wait", params.MaxWait)
	params.DeadlineMargin = n.durationOr(m, "deadline_margin", params.DeadlineMargin)

	policy, err := operation.NewSchedulerPolicy(params)
	if err != nil {
		n.add(configdecode.CodeInvalidValue, m.Pos, err.Error())
		return def
	}
	return policy
}

func (n *normalizer) deduplicationPolicy(node *configdecode.Node) operation.DeduplicationPolicy {
	m, ok := n.mapping(node, "deduplication")
	if !ok {
		return operation.DefaultDeduplicationPolicy()
	}
	configdecode.CheckUnknownFields(m, dedupFields, &n.diags)
	params := operation.DeduplicationParams{Mode: defaultDeduplicationMode}
	if s, vn, ok := n.str(m, "mode"); ok {
		v, err := operation.ParseDeduplicationMode(s)
		if err != nil {
			n.add(configdecode.CodeInvalidEnum, vn.Pos, fmt.Sprintf("invalid deduplication mode %q", s))
		} else {
			params.Mode = v
		}
	}
	params.InFlight = n.boolOr(m, "inflight", false)
	params.ScopeMemoization = n.boolOr(m, "scope_memoization", false)
	params.NegativeMemoization = n.boolOr(m, "negative_memoization", false)
	params.ErrorMemoization = n.boolOr(m, "error_memoization", false)
	params.MaxItems = int(n.intOr(m, "max_items", 0))
	params.MaxBytes = n.byteSizeOr(m, "max_bytes", 0)
	if s, vn, ok := n.str(m, "canonicalizer"); ok {
		sym, err := operation.ParseSymbol(s)
		if err != nil {
			n.add(configdecode.CodeInvalidValue, vn.Pos, fmt.Sprintf("invalid canonicalizer %q: %v", s, err))
		} else {
			params.Canonicalizer = sym
		}
	}
	policy, err := operation.NewDeduplicationPolicy(params)
	if err != nil {
		n.add(configdecode.CodeInvalidValue, m.Pos, err.Error())
		return operation.DefaultDeduplicationPolicy()
	}
	return policy
}

func (n *normalizer) retryPolicy(node *configdecode.Node) operation.RetryPolicy {
	m, ok := n.mapping(node, "retry")
	if !ok {
		return operation.DefaultRetryPolicy()
	}
	configdecode.CheckUnknownFields(m, retryFields, &n.diags)
	params := operation.RetryParams{}
	params.Enabled = n.boolOr(m, "enabled", false)
	params.MaxAttempts = int(n.intOr(m, "maximum_attempts", 0))
	params.InitialBackoff = n.durationOr(m, "initial_backoff", 0)
	params.MaxBackoff = n.durationOr(m, "maximum_backoff", 0)
	params.RespectRetryAfter = n.boolOr(m, "respect_retry_after", false)
	params.PartialItemRetry = n.boolOr(m, "partial_item_retry", false)
	if seq, _, ok := m.Get("retryable"); ok && seq.IsSequence() {
		for _, e := range seq.Elems {
			s, ok := configdecode.AsString(e)
			if !ok {
				n.add(configdecode.CodeTypeMismatch, e.Pos, "retry classification must be a string")
				continue
			}
			cls, err := operation.ParseRetryClassification(s)
			if err != nil {
				n.add(configdecode.CodeInvalidEnum, e.Pos, fmt.Sprintf("invalid retry classification %q", s))
				continue
			}
			params.Retryable = append(params.Retryable, cls)
		}
	}
	policy, err := operation.NewRetryPolicy(params)
	if err != nil {
		n.add(configdecode.CodeInvalidValue, m.Pos, err.Error())
		return operation.DefaultRetryPolicy()
	}
	return policy
}

func (n *normalizer) fallbackPolicy(node *configdecode.Node) operation.FallbackPolicy {
	m, ok := n.mapping(node, "fallback")
	if !ok {
		return operation.DefaultFallbackPolicy()
	}
	configdecode.CheckUnknownFields(m, fallbackFields, &n.diags)
	params := operation.FallbackParams{Mode: defaultFallbackMode}
	if s, vn, ok := n.str(m, "mode"); ok {
		v, err := operation.ParseFallbackMode(s)
		if err != nil {
			n.add(configdecode.CodeInvalidEnum, vn.Pos, fmt.Sprintf("invalid fallback mode %q", s))
		} else {
			params.Mode = v
		}
	}
	params.MaxScalarConcurrency = int(n.intOr(m, "max_scalar_concurrency", 0))
	params.OnQueueOverflow = n.boolOr(m, "on_queue_overflow", false)
	params.OnProviderUnavailable = n.boolOr(m, "on_provider_unavailable", false)
	params.OnUnsupportedPartition = n.boolOr(m, "on_unsupported_partition", false)
	policy, err := operation.NewFallbackPolicy(params)
	if err != nil {
		n.add(configdecode.CodeInvalidValue, m.Pos, err.Error())
		return operation.DefaultFallbackPolicy()
	}
	return policy
}

// extensions builds preserved extension data from a namespaced mapping.
func (n *normalizer) extensions(node *configdecode.Node, key string) []operation.Extension {
	m, ok := n.mapping(node, key)
	if !ok {
		return nil
	}
	var out []operation.Extension
	for _, e := range m.Entries {
		ext := operation.Extension{Namespace: e.Key, Data: encodeNodeJSON(e.Value)}
		if err := ext.Validate(); err != nil {
			n.add(configdecode.CodeInvalidValue, e.KeyPos, fmt.Sprintf("invalid extension namespace %q: %v", e.Key, err))
			continue
		}
		out = append(out, ext)
	}
	sort.Slice(out, func(i, j int) bool { return out[i].Namespace < out[j].Namespace })
	return out
}

// --- typed field readers ---

func (n *normalizer) mapping(parent *configdecode.Node, key string) (*configdecode.Node, bool) {
	v, _, ok := parent.Get(key)
	if !ok || v.IsNull() {
		return nil, false
	}
	if !v.IsMapping() {
		n.add(configdecode.CodeTypeMismatch, v.Pos, fmt.Sprintf("field %q must be a mapping, got %s", key, configdecode.TypeName(v)))
		return nil, false
	}
	return v, true
}

func (n *normalizer) str(parent *configdecode.Node, key string) (string, *configdecode.Node, bool) {
	v, _, ok := parent.Get(key)
	if !ok || v.IsNull() {
		return "", nil, false
	}
	s, ok := configdecode.AsString(v)
	if !ok {
		n.add(configdecode.CodeTypeMismatch, v.Pos, fmt.Sprintf("field %q must be a string, got %s", key, configdecode.TypeName(v)))
		return "", nil, false
	}
	return s, v, true
}

func (n *normalizer) boolField(parent *configdecode.Node, key string) (bool, bool) {
	v, _, ok := parent.Get(key)
	if !ok || v.IsNull() {
		return false, false
	}
	b, ok := configdecode.AsBool(v)
	if !ok {
		n.add(configdecode.CodeTypeMismatch, v.Pos, fmt.Sprintf("field %q must be a boolean, got %s", key, configdecode.TypeName(v)))
		return false, false
	}
	return b, true
}

func (n *normalizer) boolOr(parent *configdecode.Node, key string, def bool) bool {
	if b, ok := n.boolField(parent, key); ok {
		return b
	}
	return def
}

func (n *normalizer) intOr(parent *configdecode.Node, key string, def int) int {
	v, _, ok := parent.Get(key)
	if !ok || v.IsNull() {
		return def
	}
	iv, ok := configdecode.AsInt(v)
	if !ok {
		n.add(configdecode.CodeTypeMismatch, v.Pos, fmt.Sprintf("field %q must be an integer, got %s", key, configdecode.TypeName(v)))
		return def
	}
	return int(iv)
}

func (n *normalizer) durationOr(parent *configdecode.Node, key string, def time.Duration) time.Duration {
	s, vn, ok := n.str(parent, key)
	if !ok {
		return def
	}
	d, err := ParseDuration(s)
	if err != nil {
		n.add(configdecode.CodeInvalidDuration, vn.Pos, fmt.Sprintf("invalid duration %q for %q", s, key))
		return def
	}
	return d.Std()
}

func (n *normalizer) byteSizeOr(parent *configdecode.Node, key string, def int64) int64 {
	s, vn, ok := n.str(parent, key)
	if !ok {
		return def
	}
	b, err := ParseByteSize(s)
	if err != nil {
		n.add(configdecode.CodeInvalidByteSize, vn.Pos, fmt.Sprintf("invalid byte size %q for %q", s, key))
		return def
	}
	return b.Bytes()
}

// positionOf returns the position of node, or a zero position for nil.
func positionOf(node *configdecode.Node) diagnostics.Position {
	if node == nil {
		return diagnostics.Position{}
	}
	return node.Pos
}
