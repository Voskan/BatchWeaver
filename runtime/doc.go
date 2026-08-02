// Package runtime is BatchWeaver's explicit, typed request-coalescing runtime.
//
// It provides an instance-scoped [Engine] to which typed operations are bound
// with [Bind]. Within an explicit [Scope] carried through context, compatible
// calls to a [BoundOperation.Do] are coalesced into bounded batch provider calls
// while preserving scope isolation, partition boundaries, independent
// cancellation, caller deadlines, per-item outcomes, and deterministic lifecycle
// behavior.
//
// Applications may invoke the runtime explicitly. BatchWeaver's compiler can
// also discover proven call sites and lower them through the typed bridge
// package to this runtime without reflection. The runtime does not itself scan
// or rewrite source and never synthesizes a provider.
//
// # Design
//
// Each bound operation runs a single coordinator goroutine that owns all of its
// mutable state and processes submissions, cancellations, completions, and
// flush/drain/close control serially, so that state needs no locks. Provider
// calls run on bounded worker goroutines, never while the coordinator is
// processing an event, and never carry an individual caller's context. Callers
// select on their own result channel and their context, so one caller's
// cancellation or deadline never affects another sharing the same batch.
//
// Because this package name shadows the standard library runtime package,
// import it with an alias, for example:
//
//	import batchruntime "github.com/Voskan/BatchWeaver/runtime"
//
// Dependency direction: runtime may import the root batchweaver package,
// operation, diagnostics, and the standard library. It must not import config or
// the CLI.
package runtime
