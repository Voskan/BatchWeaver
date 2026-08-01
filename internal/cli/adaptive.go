package cli

import (
	"context"
	"encoding/json"
	"flag"
	"fmt"
	"os"
	"time"

	"github.com/Voskan/BatchWeaver/internal/adaptive"
)

// defaultBackend is the modeled backend used by the profile/tune/replay
// demonstrations when no measured backend model is supplied.
func defaultBackend() adaptive.BackendModel {
	return adaptive.BackendModel{FixedNanos: 500_000, PerItemNanos: 20_000}
}

// demoWorkload builds a deterministic synthetic workload for demonstrations.
func demoWorkload(op string, count int, rate float64) adaptive.WorkloadSpec {
	return adaptive.WorkloadSpec{
		Pattern:          adaptive.PatternPoisson,
		Operation:        op,
		Count:            count,
		RatePerSec:       rate,
		Seed:             1,
		DuplicationRate:  0.2,
		DistinctKeys:     count / 2,
		TenantClasses:    3,
		DeadlineFraction: 0.5,
		DeadlineNanos:    int64(2 * time.Millisecond),
		PayloadBytes:     256,
	}
}

// newProfileCommand returns the "profile" command.
func newProfileCommand() *Command {
	return &Command{
		Name:    "profile",
		Summary: "Collect, inspect, merge, validate, and redact workload profiles",
		Usage:   "profile collect|inspect|merge|validate|redact [flags]",
		Run:     runProfile,
	}
}

func runProfile(_ context.Context, app *App, args []string) error {
	if len(args) == 0 {
		return &CommandError{Code: ExitUsage, Message: "profile requires a subcommand: collect, inspect, merge, validate, redact"}
	}
	switch args[0] {
	case "collect":
		return profileCollect(app, args[1:])
	case "inspect":
		return profileInspect(app, args[1:])
	case "merge":
		return profileMerge(app, args[1:])
	case "validate":
		return profileValidate(app, args[1:])
	case "redact":
		return profileRedact(app, args[1:])
	default:
		return &CommandError{Code: ExitUsage, Message: fmt.Sprintf("unknown profile subcommand %q", args[0])}
	}
}

func profileCollect(app *App, args []string) error {
	fs := flag.NewFlagSet("profile collect", flag.ContinueOnError)
	fs.SetOutput(app.Stderr())
	out := fs.String("output", "", "path to write the profile bundle (.bwp)")
	op := fs.String("operation", "users.get", "operation ID for the synthetic workload")
	count := fs.Int("count", 5000, "number of synthetic calls")
	rate := fs.Float64("rate", 8000, "synthetic arrival rate per second")
	synthetic := fs.Bool("synthetic", true, "collect from a deterministic synthetic workload (no live app in this build)")
	if err := fs.Parse(args); err != nil {
		return &CommandError{Code: ExitUsage}
	}
	if !*synthetic {
		return &CommandError{Code: ExitUsage, Message: "this build collects from --synthetic workloads; attach the collector to a live runtime in production"}
	}
	settings := adaptive.Settings{MaxWaitNanos: int64(500 * time.Microsecond), MaxBatchSize: 256, MaxConcurrency: 8}
	bundle := adaptive.CollectSynthetic(demoWorkload(*op, *count, *rate), settings, defaultBackend())
	w := app.Stdout()
	fmt.Fprintln(w, "BatchWeaver profile collection (synthetic)")
	fmt.Fprintf(w, "\nOperations observed:   %d\n", len(bundle.Operations))
	if p, ok := bundle.FindOperation(*op); ok {
		fmt.Fprintf(w, "Logical calls:         %d\n", p.Arrivals.LogicalCalls)
		fmt.Fprintf(w, "Backend calls:         %d\n", p.Batches.BackendCalls)
	}
	fmt.Fprintf(w, "Redacted fields:       %d\n", bundle.Redaction.RedactedFields)
	fmt.Fprintf(w, "Raw keys stored:       %t\n", bundle.Redaction.RawKeysStored)
	fmt.Fprintf(w, "Profile digest:        %s\n", bundle.Digest.Short())
	if *out != "" {
		if err := adaptive.WriteFile(*out, bundle); err != nil {
			return &CommandError{Code: ExitError, Message: err.Error()}
		}
		fmt.Fprintf(w, "\nProfile bundle:\n  %s\n", *out)
	}
	fmt.Fprintln(w, "\nNo raw operation keys were stored.")
	return nil
}

func profileInspect(app *App, args []string) error {
	bundle, cerr := readProfileFlag("profile inspect", app, args)
	if cerr != nil {
		return cerr
	}
	w := app.Stdout()
	fmt.Fprintf(w, "Profile bundle:\n  id:      %s\n  digest:  %s\n  schema:  %s\n  abi:     %s\n\n",
		bundle.ID, bundle.Digest.Short(), bundle.SchemaVersion, bundle.RuntimeABI)
	fmt.Fprintf(w, "Operations: %d\n", len(bundle.Operations))
	for i := range bundle.Operations {
		p := &bundle.Operations[i]
		fmt.Fprintf(w, "\n%s\n", p.Operation)
		fmt.Fprintf(w, "  logical calls:   %d\n", p.Arrivals.LogicalCalls)
		fmt.Fprintf(w, "  backend calls:   %d\n", p.Batches.BackendCalls)
		fmt.Fprintf(w, "  duplicate rate:  %.3f\n", ratef(p.Duplicates.Duplicates, p.Duplicates.Duplicates+p.Duplicates.Unique))
		fmt.Fprintf(w, "  backend fixed:   %s\n", time.Duration(p.Backend.FixedCostNanos))
		fmt.Fprintf(w, "  partition classes: %d\n", p.Partitions.DistinctClasses)
	}
	return nil
}

func profileMerge(app *App, args []string) error {
	fs := flag.NewFlagSet("profile merge", flag.ContinueOnError)
	fs.SetOutput(app.Stderr())
	out := fs.String("output", "", "path to write the merged profile")
	if err := fs.Parse(args); err != nil {
		return &CommandError{Code: ExitUsage}
	}
	files := fs.Args()
	if len(files) < 2 {
		return &CommandError{Code: ExitUsage, Message: "profile merge requires at least two profile paths"}
	}
	var bundles []*adaptive.ProfileBundle
	for _, f := range files {
		b, err := adaptive.ReadFile(f)
		if err != nil {
			return &CommandError{Code: ExitError, Message: err.Error()}
		}
		bundles = append(bundles, b)
	}
	merged, err := adaptive.Merge(bundles...)
	if err != nil {
		return &CommandError{Code: ExitConfigInvalid, Message: err.Error()}
	}
	w := app.Stdout()
	fmt.Fprintf(w, "Merged %d profiles into one bundle: %s\n", len(bundles), merged.Digest.Short())
	if *out != "" {
		if err := adaptive.WriteFile(*out, merged); err != nil {
			return &CommandError{Code: ExitError, Message: err.Error()}
		}
		fmt.Fprintf(w, "Written to:\n  %s\n", *out)
	}
	return nil
}

func profileValidate(app *App, args []string) error {
	fs := flag.NewFlagSet("profile validate", flag.ContinueOnError)
	fs.SetOutput(app.Stderr())
	file := fs.String("profile", "", "path to a profile bundle")
	abi := fs.String("abi", "", "expected runtime ABI (optional)")
	maxAge := fs.Duration("max-age", 0, "maximum age for active use (optional)")
	if err := fs.Parse(args); err != nil {
		return &CommandError{Code: ExitUsage}
	}
	if *file == "" {
		return &CommandError{Code: ExitUsage, Message: "profile validate requires --profile"}
	}
	bundle, err := adaptive.ReadFile(*file)
	if err != nil {
		return &CommandError{Code: ExitError, Message: err.Error()}
	}
	res := adaptive.CheckCompatibility(bundle, adaptive.CompatibilityRequirement{RuntimeABI: *abi, MaxAge: *maxAge})
	w := app.Stdout()
	fmt.Fprintf(w, "Profile:\n  %s\n\n", *file)
	fmt.Fprintf(w, "Compatible: %t\nStale:      %t\n", res.Compatible, res.Stale)
	for _, d := range res.Diagnostics {
		fmt.Fprintf(w, "  %s %s: %s\n", d.Code, d.Severity, d.Detail)
	}
	if !res.Compatible {
		return &CommandError{Code: ExitConfigInvalid}
	}
	if res.Stale {
		return &CommandError{Code: ExitStale}
	}
	return nil
}

func profileRedact(app *App, args []string) error {
	bundle, cerr := readProfileFlag("profile redact", app, args)
	if cerr != nil {
		return cerr
	}
	w := app.Stdout()
	fmt.Fprintln(w, "Profile redaction summary")
	fmt.Fprintf(w, "\n  partition classing: %s\n  tenant classing:    %s\n  redacted fields:    %d\n  raw keys stored:    %t\n",
		bundle.Redaction.PartitionClassing, bundle.Redaction.TenantClassing,
		bundle.Redaction.RedactedFields, bundle.Redaction.RawKeysStored)
	if bundle.Redaction.RawKeysStored {
		fmt.Fprintln(w, "\nWARNING: this profile claims to store raw keys; this must never happen.")
		return &CommandError{Code: ExitConfigInvalid}
	}
	fmt.Fprintln(w, "\nNo raw operation keys, tokens, tenants, or payloads are present.")
	return nil
}

// readProfileFlag parses a --profile flag and loads the bundle.
func readProfileFlag(name string, app *App, args []string) (*adaptive.ProfileBundle, *CommandError) {
	fs := flag.NewFlagSet(name, flag.ContinueOnError)
	fs.SetOutput(app.Stderr())
	file := fs.String("profile", "", "path to a profile bundle")
	if err := fs.Parse(args); err != nil {
		return nil, &CommandError{Code: ExitUsage}
	}
	if *file == "" {
		return nil, &CommandError{Code: ExitUsage, Message: name + " requires --profile"}
	}
	b, err := adaptive.ReadFile(*file)
	if err != nil {
		return nil, &CommandError{Code: ExitError, Message: err.Error()}
	}
	return b, nil
}

func ratef(num, den uint64) float64 {
	if den == 0 {
		return 0
	}
	return float64(num) / float64(den)
}

// --- tune ---

// newTuneCommand returns the "tune" command.
func newTuneCommand() *Command {
	return &Command{
		Name:    "tune",
		Summary: "Analyze, shadow, replay, explain, and report adaptive tuning",
		Usage:   "tune analyze|shadow|replay|explain|report|freeze|rollback [flags]",
		Run:     runTune,
	}
}

func runTune(_ context.Context, app *App, args []string) error {
	if len(args) == 0 {
		return &CommandError{Code: ExitUsage, Message: "tune requires a subcommand"}
	}
	switch args[0] {
	case "analyze", "report":
		return tuneAnalyze(app, args[0], args[1:])
	case "shadow":
		return tuneShadow(app, args[1:])
	case "replay":
		return tuneReplay(app, args[1:])
	case "explain":
		return tuneExplain(app, args[1:])
	case "freeze":
		return tuneFreeze(app)
	case "rollback":
		return tuneRollback(app)
	default:
		return &CommandError{Code: ExitUsage, Message: fmt.Sprintf("unknown tune subcommand %q", args[0])}
	}
}

func buildController(objective string, mode adaptive.TuningMode) *adaptive.Controller {
	obj := adaptive.ObjectivePolicy(objective)
	if !adaptive.KnownObjective(obj) {
		obj = adaptive.ObjectiveBalanced
	}
	return adaptive.NewController(adaptive.ControllerConfig{
		Mode:      mode,
		Objective: obj,
		Bounds:    adaptive.DefaultBounds(),
		Clock:     adaptive.SystemClock(),
	})
}

func currentSettingsMap(bundle *adaptive.ProfileBundle) map[string]adaptive.Settings {
	m := map[string]adaptive.Settings{}
	for i := range bundle.Operations {
		m[bundle.Operations[i].Operation] = adaptive.Settings{
			MaxWaitNanos: int64(500 * time.Microsecond), MaxBatchSize: 256, MaxConcurrency: 8,
		}
	}
	return m
}

func tuneAnalyze(app *App, sub string, args []string) error {
	fs := flag.NewFlagSet("tune "+sub, flag.ContinueOnError)
	fs.SetOutput(app.Stderr())
	file := fs.String("profile", "", "path to a profile bundle")
	objective := fs.String("objective", "balanced", "objective policy")
	format := fs.String("format", "text", "output format: text, json, or markdown")
	out := fs.String("output", "", "optional path to write the report")
	if err := fs.Parse(args); err != nil {
		return &CommandError{Code: ExitUsage}
	}
	if *file == "" {
		return &CommandError{Code: ExitUsage, Message: "tune " + sub + " requires --profile"}
	}
	bundle, err := adaptive.ReadFile(*file)
	if err != nil {
		return &CommandError{Code: ExitError, Message: err.Error()}
	}
	ctrl := buildController(*objective, adaptive.TuningShadow)
	rep := adaptive.AnalyzeBundle(ctrl, bundle, currentSettingsMap(bundle), 0)
	var rendered string
	switch *format {
	case "json":
		b, err := rep.JSON()
		if err != nil {
			return &CommandError{Code: ExitError, Message: err.Error()}
		}
		rendered = string(b) + "\n"
	case "markdown", "md":
		rendered = rep.Markdown()
	default:
		rendered = rep.Text()
	}
	if *out != "" {
		if err := os.WriteFile(*out, []byte(rendered), 0o644); err != nil {
			return &CommandError{Code: ExitError, Message: err.Error()}
		}
		fmt.Fprintf(app.Stdout(), "Report written to:\n  %s\n", *out)
		return nil
	}
	fmt.Fprint(app.Stdout(), rendered)
	return nil
}

func tuneShadow(app *App, args []string) error {
	fs := flag.NewFlagSet("tune shadow", flag.ContinueOnError)
	fs.SetOutput(app.Stderr())
	file := fs.String("profile", "", "path to a profile bundle")
	objective := fs.String("objective", "balanced", "objective policy")
	if err := fs.Parse(args); err != nil {
		return &CommandError{Code: ExitUsage}
	}
	if *file == "" {
		return &CommandError{Code: ExitUsage, Message: "tune shadow requires --profile"}
	}
	bundle, err := adaptive.ReadFile(*file)
	if err != nil {
		return &CommandError{Code: ExitError, Message: err.Error()}
	}
	ctrl := buildController(*objective, adaptive.TuningShadow)
	cur := currentSettingsMap(bundle)
	w := app.Stdout()
	fmt.Fprintln(w, "Shadow tuning (no runtime settings are changed)")
	for i := range bundle.Operations {
		op := &bundle.Operations[i]
		d := ctrl.Recommend(adaptive.RecommendInput{Profile: op, Current: cur[op.Operation], ProfileDigest: op.Digest})
		fmt.Fprintf(w, "\n%s [%s]\n", op.Operation, d.ID)
		fmt.Fprintf(w, "  would set max_wait: %s (from %s)\n", d.New.MaxWait(), d.Previous.MaxWait())
		fmt.Fprintf(w, "  confidence:         %s\n", d.ConfidenceLabel)
	}
	return nil
}

func tuneReplay(app *App, args []string) error {
	fs := flag.NewFlagSet("tune replay", flag.ContinueOnError)
	fs.SetOutput(app.Stderr())
	op := fs.String("operation", "users.get", "operation for the synthetic replay workload")
	count := fs.Int("count", 5000, "number of synthetic calls")
	rate := fs.Float64("rate", 8000, "arrival rate per second")
	if err := fs.Parse(args); err != nil {
		return &CommandError{Code: ExitUsage}
	}
	events := adaptive.GenerateWorkload(demoWorkload(*op, *count, *rate))
	backend := defaultBackend()
	cost := adaptive.NewCostModel(adaptive.ObjectiveBalanced, adaptive.CostWeights{})
	sims := []adaptive.PolicySim{
		{Name: "current", Settings: adaptive.Settings{MaxWaitNanos: int64(500 * time.Microsecond), MaxBatchSize: 256, MaxConcurrency: 8}},
		{Name: "latency", Settings: adaptive.Settings{MaxWaitNanos: int64(100 * time.Microsecond), MaxBatchSize: 64, MaxConcurrency: 8}},
		{Name: "throughput", Settings: adaptive.Settings{MaxWaitNanos: int64(2 * time.Millisecond), MaxBatchSize: 512, MaxConcurrency: 8}},
	}
	cmp := adaptive.ComparePolicies(events, backend, cost, sims...)
	w := app.Stdout()
	fmt.Fprintf(w, "Deterministic replay over %d events (model estimates)\n\n", cmp.Events)
	for _, r := range cmp.Results {
		fmt.Fprintf(w, "%-12s backend_calls=%-6d mean_batch=%-6.1f p95_queue=%-10s deadline_misses=%-5d cost=%.0f\n",
			r.Policy, r.BackendCalls, r.MeanBatchSize, time.Duration(r.P95QueueDelayNanos), r.DeadlineMisses, r.CostScore)
	}
	return nil
}

func tuneExplain(app *App, args []string) error {
	fs := flag.NewFlagSet("tune explain", flag.ContinueOnError)
	fs.SetOutput(app.Stderr())
	file := fs.String("profile", "", "path to a profile bundle")
	operation := fs.String("operation", "", "operation to explain")
	objective := fs.String("objective", "balanced", "objective policy")
	if err := fs.Parse(args); err != nil {
		return &CommandError{Code: ExitUsage}
	}
	if *file == "" || *operation == "" {
		return &CommandError{Code: ExitUsage, Message: "tune explain requires --profile and --operation"}
	}
	bundle, err := adaptive.ReadFile(*file)
	if err != nil {
		return &CommandError{Code: ExitError, Message: err.Error()}
	}
	p, ok := bundle.FindOperation(*operation)
	if !ok {
		return &CommandError{Code: ExitConfigInvalid, Message: "operation not found in profile"}
	}
	ctrl := buildController(*objective, adaptive.TuningShadow)
	cur := adaptive.Settings{MaxWaitNanos: int64(500 * time.Microsecond), MaxBatchSize: 256, MaxConcurrency: 8}
	d := ctrl.Recommend(adaptive.RecommendInput{Profile: p, Current: cur, ProfileDigest: p.Digest})
	w := app.Stdout()
	fmt.Fprintf(w, "Decision:\n  %s\n\n", d.ID)
	fmt.Fprintf(w, "Change:\n  max_wait %s -> %s, max_batch_size %d -> %d\n\n",
		d.Previous.MaxWait(), d.New.MaxWait(), d.Previous.MaxBatchSize, d.New.MaxBatchSize)
	fmt.Fprintln(w, "Reasons:")
	for _, r := range d.Reasons {
		fmt.Fprintf(w, "  - %s\n", r)
	}
	fmt.Fprintf(w, "\nEvidence:\n  arrival rate:    %.1f/s\n  duplicate rate:  %.3f\n  p95 queue delay: %s\n  p95 backend:     %s\n",
		d.Evidence.ArrivalRatePerSec, d.Evidence.DuplicateRate,
		time.Duration(d.Evidence.P95QueueDelayNanos), time.Duration(d.Evidence.P95BackendNanos))
	fmt.Fprintf(w, "\nConfidence:\n  %s\n", d.ConfidenceLabel)
	fmt.Fprintf(w, "\nSafety:\n  within configured bounds\n  applied: %t\n", d.Applied)
	if len(d.Diagnostics) > 0 {
		fmt.Fprintln(w, "\nDiagnostics:")
		for _, dg := range d.Diagnostics {
			fmt.Fprintf(w, "  %s %s: %s\n", dg.Code, dg.Severity, dg.Detail)
		}
	}
	return nil
}

func tuneFreeze(app *App) error {
	w := app.Stdout()
	fmt.Fprintln(w, "Emergency freeze")
	fmt.Fprintln(w, "\nIn active mode the runtime calls BoundOperation.ClearAdaptiveSettings and the")
	fmt.Fprintln(w, "controller EmergencyDisable, restoring configured defaults immediately. This")
	fmt.Fprintln(w, "path does not depend on the profile store being available.")
	return nil
}

func tuneRollback(app *App) error {
	w := app.Stdout()
	fmt.Fprintln(w, "Manual rollback")
	fmt.Fprintln(w, "\nThe RollbackMonitor restores the settings in effect before the last applied")
	fmt.Fprintln(w, "decision and records a BW8006 diagnostic. Automatic rollback also fires when an")
	fmt.Fprintln(w, "SLO guardrail is breached within the rollback window.")
	return nil
}

// --- fairness ---

// newFairnessCommand returns the "fairness" command.
func newFairnessCommand() *Command {
	return &Command{
		Name:    "fairness",
		Summary: "Report fairness across anonymized classes",
		Usage:   "fairness report [--operation=op]",
		Run:     runFairness,
	}
}

func runFairness(_ context.Context, app *App, args []string) error {
	if len(args) == 0 || args[0] != "report" {
		return &CommandError{Code: ExitUsage, Message: "usage: fairness report [--operation=op]"}
	}
	fs := flag.NewFlagSet("fairness report", flag.ContinueOnError)
	fs.SetOutput(app.Stderr())
	operation := fs.String("operation", "users.get", "operation to report on")
	if err := fs.Parse(args[1:]); err != nil {
		return &CommandError{Code: ExitUsage}
	}
	// Deterministic demonstration across three anonymized classes.
	clock := adaptive.NewFakeClock(time.Unix(0, 0))
	sched := adaptive.NewFairScheduler(adaptive.FairnessConfig{
		Algorithm:           adaptive.FairWeighted,
		StarvationThreshold: 100 * time.Millisecond,
		Classes: []adaptive.ClassPolicy{
			{Class: "class_a", Weight: 3, Priority: 1, ReservedShare: 0.2},
			{Class: "class_b", Weight: 1},
			{Class: "class_c", Weight: 1},
		},
	}, clock)
	for i := 0; i < 300; i++ {
		sched.Admit("class_a", 0)
		sched.Admit("class_b", 0)
		if i%3 == 0 {
			sched.Admit("class_c", 0)
		}
	}
	for i := 0; i < 400; i++ {
		if class, ok := sched.Next(); ok {
			sched.Serve(class)
		}
	}
	rep := sched.Report()
	w := app.Stdout()
	fmt.Fprintf(w, "Fairness report (%s, algorithm=%s)\n\n", *operation, rep.Algorithm)
	for _, c := range rep.Classes {
		fmt.Fprintf(w, "  %-10s share=%.3f served=%-5d queued=%-4d reserved=%.2f starved=%t\n",
			c.Class, c.ServiceShare, c.Served, c.Queued, c.Reserved, c.Starved)
	}
	fmt.Fprintln(w, "\nTenant identities are anonymized to classes; raw IDs are never shown.")
	return nil
}

// --- overload ---

// newOverloadCommand returns the "overload" command.
func newOverloadCommand() *Command {
	return &Command{
		Name:    "overload",
		Summary: "Inspect overload state and admission decisions",
		Usage:   "overload inspect [--queue=0.9 --timeout-rate=0.0 --policy=shed-low-priority]",
		Run:     runOverload,
	}
}

func runOverload(_ context.Context, app *App, args []string) error {
	if len(args) == 0 || args[0] != "inspect" {
		return &CommandError{Code: ExitUsage, Message: "usage: overload inspect [flags]"}
	}
	fs := flag.NewFlagSet("overload inspect", flag.ContinueOnError)
	fs.SetOutput(app.Stderr())
	queue := fs.Float64("queue", 0.9, "queue depth ratio in [0,1+]")
	timeout := fs.Float64("timeout-rate", 0, "timeout rate in [0,1]")
	throttle := fs.Float64("throttle-rate", 0, "throttle rate in [0,1]")
	policy := fs.String("policy", "shed-low-priority", "admission policy")
	if err := fs.Parse(args[1:]); err != nil {
		return &CommandError{Code: ExitUsage}
	}
	det := adaptive.NewOverloadDetector(adaptive.OverloadConfig{Policy: adaptive.AdmissionPolicy(*policy)})
	sig := adaptive.OverloadSignals{QueueDepthRatio: *queue, TimeoutRate: *timeout, ThrottleRate: *throttle}
	w := app.Stdout()
	fmt.Fprintf(w, "Overload state:\n  %s\n\n", det.State(sig))
	fmt.Fprintln(w, "Admission decisions:")
	for _, req := range []struct {
		name string
		r    adaptive.AdmissionRequest
	}{
		{"normal request", adaptive.AdmissionRequest{HasFallback: true}},
		{"low-priority request", adaptive.AdmissionRequest{LowPriority: true, HasFallback: true}},
		{"critical request", adaptive.AdmissionRequest{Critical: true}},
	} {
		d := det.Admit(req.r, sig)
		fmt.Fprintf(w, "  %-22s -> %s (%s)\n", req.name, d.Action, d.Reason)
	}
	bp := det.Backpressure(sig, int64(250*time.Microsecond), 0)
	fmt.Fprintf(w, "\nBackpressure:\n  overloaded=%t queue_full=%t recommend_direct=%t\n", bp.Overloaded, bp.QueueFull, bp.RecommendDirect)
	return nil
}

// --- wave ---

// newWaveCommand returns the "wave" command.
func newWaveCommand() *Command {
	return &Command{
		Name:    "wave",
		Summary: "Build and render multi-operation wave graphs",
		Usage:   "wave graph [--file=graph.json] [--format=dot|text]",
		Run:     runWave,
	}
}

// waveSpec is the JSON input for a wave graph.
type waveSpec struct {
	Nodes []adaptive.Node `json:"nodes"`
	Edges []adaptive.Edge `json:"edges"`
}

func runWave(_ context.Context, app *App, args []string) error {
	if len(args) == 0 || args[0] != "graph" {
		return &CommandError{Code: ExitUsage, Message: "usage: wave graph [--file=graph.json] [--format=dot|text]"}
	}
	fs := flag.NewFlagSet("wave graph", flag.ContinueOnError)
	fs.SetOutput(app.Stderr())
	file := fs.String("file", "", "path to a wave graph JSON spec")
	format := fs.String("format", "text", "output format: text or dot")
	if err := fs.Parse(args[1:]); err != nil {
		return &CommandError{Code: ExitUsage}
	}
	g := adaptive.NewWaveGraph()
	if *file != "" {
		data, err := os.ReadFile(*file)
		if err != nil {
			return &CommandError{Code: ExitError, Message: err.Error()}
		}
		var spec waveSpec
		if err := json.Unmarshal(data, &spec); err != nil {
			return &CommandError{Code: ExitConfigInvalid, Message: err.Error()}
		}
		for _, n := range spec.Nodes {
			if err := g.AddNode(n); err != nil {
				return &CommandError{Code: ExitConfigInvalid, Message: err.Error()}
			}
		}
		for _, e := range spec.Edges {
			if err := g.AddEdge(e); err != nil {
				return &CommandError{Code: ExitConfigInvalid, Message: err.Error()}
			}
		}
	} else {
		buildDemoWaveGraph(g)
	}
	if diag := g.Validate(); diag != nil {
		fmt.Fprintf(app.Stdout(), "Wave graph invalid\n\n%s: %s\n", diag.Code, diag.Detail)
		return &CommandError{Code: ExitConfigInvalid}
	}
	w := app.Stdout()
	if *format == "dot" {
		fmt.Fprint(w, g.DOT())
		return nil
	}
	waves, _ := g.Waves()
	fmt.Fprintln(w, "Dispatch waves:")
	for _, wv := range waves {
		fmt.Fprintf(w, "  wave %d: %v\n", wv.Level, wv.Nodes)
		for grp, members := range wv.FusionGroups {
			fmt.Fprintf(w, "    fusion group %q: %v\n", grp, members)
		}
	}
	fmt.Fprintf(w, "\nCritical path:\n  %v\n", g.CriticalPath())
	return nil
}

func buildDemoWaveGraph(g *adaptive.WaveGraph) {
	_ = g.AddNode(adaptive.Node{ID: "load_user", Kind: adaptive.NodeOperation, Operation: "users.get", Cost: 3})
	_ = g.AddNode(adaptive.Node{ID: "load_org", Kind: adaptive.NodeOperation, Operation: "orgs.get", Cost: 2})
	_ = g.AddNode(adaptive.Node{ID: "load_perms", Kind: adaptive.NodeOperation, Operation: "perms.get", Cost: 4})
	_ = g.AddNode(adaptive.Node{ID: "render", Kind: adaptive.NodeComputation, Cost: 1})
	_ = g.AddEdge(adaptive.Edge{From: "load_user", To: "load_perms", Kind: adaptive.EdgeData})
	_ = g.AddEdge(adaptive.Edge{From: "load_org", To: "load_perms", Kind: adaptive.EdgeData})
	_ = g.AddEdge(adaptive.Edge{From: "load_perms", To: "render", Kind: adaptive.EdgeData})
}

// --- recursive ---

// newRecursiveCommand returns the "recursive" command.
func newRecursiveCommand() *Command {
	return &Command{
		Name:    "recursive",
		Summary: "Inspect recursive breadth-first batching",
		Usage:   "recursive inspect [--depth=N]",
		Run:     runRecursive,
	}
}

func runRecursive(ctx context.Context, app *App, args []string) error {
	if len(args) == 0 || args[0] != "inspect" {
		return &CommandError{Code: ExitUsage, Message: "usage: recursive inspect [--depth=N]"}
	}
	fs := flag.NewFlagSet("recursive inspect", flag.ContinueOnError)
	fs.SetOutput(app.Stderr())
	depth := fs.Int("depth", 4, "maximum traversal depth")
	if err := fs.Parse(args[1:]); err != nil {
		return &CommandError{Code: ExitUsage}
	}
	// Deterministic demo: a binary tree loaded breadth-first, one batch per level.
	contract := adaptive.RecursiveContract[int, int]{
		Children: func(_ int, v int) []int {
			if v >= 100 {
				return nil
			}
			return []int{v*2 + 1, v*2 + 2}
		},
		Limits:      adaptive.RecursiveLimits{MaxDepth: *depth},
		Cycle:       adaptive.CycleSkipSeen,
		ErrorPolicy: adaptive.ErrCollectPerNode,
		ProofValid:  true,
	}
	loader := func(_ context.Context, keys []int) ([]adaptive.NodeResult[int, int], error) {
		out := make([]adaptive.NodeResult[int, int], len(keys))
		for i, k := range keys {
			out[i] = adaptive.NodeResult[int, int]{Key: k, Value: k, Found: true}
		}
		return out, nil
	}
	res, err := adaptive.Traverse(ctx, []int{0}, contract, loader)
	w := app.Stdout()
	fmt.Fprintln(w, "Recursive breadth-first traversal (demonstration)")
	fmt.Fprintf(w, "\n  frontier sizes: %v\n  depth reached:  %d\n  nodes visited:  %d\n  edges:          %d\n",
		res.FrontierSizes, res.DepthReached, res.Nodes, res.Edges)
	fmt.Fprintln(w, "\nOne batched load is issued per breadth-first frontier level.")
	if err != nil {
		for _, d := range res.Diagnostics {
			fmt.Fprintf(w, "  %s: %s\n", d.Code, d.Detail)
		}
	}
	return nil
}
