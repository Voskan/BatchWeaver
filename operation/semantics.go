package operation

import (
	"errors"
	"fmt"
)

// Idempotency describes whether repeating an operation changes its observable
// outcome.
type Idempotency uint8

const (
	// IdempotencyUnknown is the zero value and is invalid in a complete spec.
	IdempotencyUnknown Idempotency = iota
	// IdempotencyIdempotent means repetition is safe.
	IdempotencyIdempotent
	// IdempotencyNonIdempotent means repetition is unsafe.
	IdempotencyNonIdempotent
	// IdempotencyConditional means repetition is safe only with an
	// idempotency-key contract, which a later runtime prompt enforces.
	IdempotencyConditional
)

// ErrInvalidIdempotency is returned for an unknown idempotency value.
var ErrInvalidIdempotency = errors.New("invalid idempotency")

var idempotencyNames = []string{"unknown", "idempotent", "non-idempotent", "conditional"}

// String returns the canonical name of the idempotency value.
func (i Idempotency) String() string { return enumString(i, idempotencyNames) }

// Valid reports whether i is defined and not Unknown.
func (i Idempotency) Valid() bool {
	return i != IdempotencyUnknown && int(i) < len(idempotencyNames)
}

// ParseIdempotency resolves the canonical name of an idempotency value.
func ParseIdempotency(s string) (Idempotency, error) {
	return parseEnum[Idempotency](s, idempotencyNames, ErrInvalidIdempotency)
}

// MarshalText implements encoding.TextMarshaler.
func (i Idempotency) MarshalText() ([]byte, error) {
	return marshalEnum(i, idempotencyNames, ErrInvalidIdempotency)
}

// UnmarshalText implements encoding.TextUnmarshaler.
func (i *Idempotency) UnmarshalText(data []byte) error {
	v, err := ParseIdempotency(string(data))
	if err != nil {
		return err
	}
	*i = v
	return nil
}

// Ordering describes whether an operation's items may be reordered.
type Ordering uint8

const (
	// OrderingPreserve keeps the caller's order; it is the safe default.
	OrderingPreserve Ordering = iota
	// OrderingReorderable permits reordering when the caller opts in.
	OrderingReorderable
	// OrderingCommutative permits reordering because order never matters.
	OrderingCommutative
)

// ErrInvalidOrdering is returned for an unknown ordering value.
var ErrInvalidOrdering = errors.New("invalid ordering")

var orderingNames = []string{"preserve", "reorderable", "commutative"}

// String returns the canonical name of the ordering value.
func (o Ordering) String() string { return enumString(o, orderingNames) }

// Valid reports whether o is a defined ordering value.
func (o Ordering) Valid() bool { return int(o) < len(orderingNames) }

// ParseOrdering resolves the canonical name of an ordering value.
func ParseOrdering(s string) (Ordering, error) {
	return parseEnum[Ordering](s, orderingNames, ErrInvalidOrdering)
}

// MarshalText implements encoding.TextMarshaler.
func (o Ordering) MarshalText() ([]byte, error) {
	return marshalEnum(o, orderingNames, ErrInvalidOrdering)
}

// UnmarshalText implements encoding.TextUnmarshaler.
func (o *Ordering) UnmarshalText(data []byte) error {
	v, err := ParseOrdering(string(data))
	if err != nil {
		return err
	}
	*o = v
	return nil
}

// Atomicity describes how partial failure within a batch is handled.
type Atomicity uint8

const (
	// AtomicityPerItem allows each item to succeed or fail independently.
	AtomicityPerItem Atomicity = iota
	// AtomicityAllOrNothing requires every item to succeed or all to fail.
	AtomicityAllOrNothing
	// AtomicityPrefix commits a successful prefix of items.
	AtomicityPrefix
	// AtomicityProviderDefined defers atomicity semantics to the provider.
	AtomicityProviderDefined
)

// ErrInvalidAtomicity is returned for an unknown atomicity value.
var ErrInvalidAtomicity = errors.New("invalid atomicity")

var atomicityNames = []string{"per-item", "all-or-nothing", "prefix", "provider-defined"}

// String returns the canonical name of the atomicity value.
func (a Atomicity) String() string { return enumString(a, atomicityNames) }

// Valid reports whether a is a defined atomicity value.
func (a Atomicity) Valid() bool { return int(a) < len(atomicityNames) }

// ParseAtomicity resolves the canonical name of an atomicity value.
func ParseAtomicity(s string) (Atomicity, error) {
	return parseEnum[Atomicity](s, atomicityNames, ErrInvalidAtomicity)
}

// MarshalText implements encoding.TextMarshaler.
func (a Atomicity) MarshalText() ([]byte, error) {
	return marshalEnum(a, atomicityNames, ErrInvalidAtomicity)
}

// UnmarshalText implements encoding.TextUnmarshaler.
func (a *Atomicity) UnmarshalText(data []byte) error {
	v, err := ParseAtomicity(string(data))
	if err != nil {
		return err
	}
	*a = v
	return nil
}

// ErrInvalidSemantics is returned when a Semantics value fails validation.
var ErrInvalidSemantics = errors.New("invalid semantics")

// Semantics is an immutable-by-convention bundle of the semantic properties
// that govern whether and how an operation may be batched. Construct it through
// a named constructor such as ReadOnly; the named constructors produce fully
// valid defaults, and the few available modifiers only permit semantically safe
// changes.
type Semantics struct {
	kind                 Kind
	effect               Effect
	idempotency          Idempotency
	ordering             Ordering
	atomicity            Atomicity
	retryAllowed         bool
	deduplicationAllowed bool
	crossScopeAllowed    bool
	freshnessDependent   bool
}

// Kind returns the operation kind.
func (s Semantics) Kind() Kind { return s.kind }

// Effect returns the externally observable effect.
func (s Semantics) Effect() Effect { return s.effect }

// Idempotency returns the idempotency classification.
func (s Semantics) Idempotency() Idempotency { return s.idempotency }

// Ordering returns the ordering requirement.
func (s Semantics) Ordering() Ordering { return s.ordering }

// Atomicity returns the atomicity requirement.
func (s Semantics) Atomicity() Atomicity { return s.atomicity }

// RetryAllowed reports whether automatic retry is semantically permitted.
func (s Semantics) RetryAllowed() bool { return s.retryAllowed }

// DeduplicationAllowed reports whether deduplication is semantically permitted.
func (s Semantics) DeduplicationAllowed() bool { return s.deduplicationAllowed }

// CrossScopeBatchingAllowed reports whether cross-scope batching may ever be
// permitted. It defaults to false and cannot be enabled implicitly.
func (s Semantics) CrossScopeBatchingAllowed() bool { return s.crossScopeAllowed }

// FreshnessDependent reports whether results depend on freshness or consistency.
func (s Semantics) FreshnessDependent() bool { return s.freshnessDependent }

// WithReorderable returns a copy that permits reordering. It is only valid for
// read-only operations; validation rejects reordering for other kinds.
func (s Semantics) WithReorderable() Semantics {
	s.ordering = OrderingReorderable
	return s
}

// Validate reports whether the semantics are internally consistent.
func (s Semantics) Validate() error {
	if !s.kind.Valid() {
		return fmt.Errorf("%w: %w", ErrInvalidSemantics, ErrInvalidKind)
	}
	if s.effect != s.kind.Effect() {
		return fmt.Errorf("%w: effect %s does not match kind %s", ErrInvalidSemantics, s.effect, s.kind)
	}
	if !s.idempotency.Valid() {
		return fmt.Errorf("%w: %w", ErrInvalidSemantics, ErrInvalidIdempotency)
	}
	if !s.ordering.Valid() {
		return fmt.Errorf("%w: %w", ErrInvalidSemantics, ErrInvalidOrdering)
	}
	if !s.atomicity.Valid() {
		return fmt.Errorf("%w: %w", ErrInvalidSemantics, ErrInvalidAtomicity)
	}
	if s.kind == KindNonIdempotentWrite && s.ordering != OrderingPreserve {
		return fmt.Errorf("%w: non-idempotent writes must preserve order", ErrInvalidSemantics)
	}
	if s.kind == KindCommutativeAggregation && s.ordering != OrderingCommutative {
		return fmt.Errorf("%w: commutative aggregation must declare commutative ordering", ErrInvalidSemantics)
	}
	if s.kind == KindOrderedAggregation && s.ordering != OrderingPreserve {
		return fmt.Errorf("%w: ordered aggregation must preserve order", ErrInvalidSemantics)
	}
	if s.kind == KindAtomicGroup && s.atomicity != AtomicityAllOrNothing {
		return fmt.Errorf("%w: atomic group must use all-or-nothing atomicity", ErrInvalidSemantics)
	}
	if s.retryAllowed && s.idempotency == IdempotencyNonIdempotent {
		return fmt.Errorf("%w: non-idempotent operations cannot allow retry", ErrInvalidSemantics)
	}
	if s.ordering == OrderingReorderable && s.effect != EffectRead {
		return fmt.Errorf("%w: only read operations may be reorderable", ErrInvalidSemantics)
	}
	if s.freshnessDependent && s.effect != EffectRead {
		return fmt.Errorf("%w: only reads may be freshness dependent", ErrInvalidSemantics)
	}
	return nil
}

// ReadOnly returns semantics for a read that observes stable state.
func ReadOnly() Semantics {
	return Semantics{
		kind: KindReadOnly, effect: EffectRead, idempotency: IdempotencyIdempotent,
		ordering: OrderingPreserve, atomicity: AtomicityPerItem,
		retryAllowed: true, deduplicationAllowed: true,
	}
}

// FreshnessSensitiveRead returns semantics for a read whose result depends on
// freshness or a consistency level.
func FreshnessSensitiveRead() Semantics {
	return Semantics{
		kind: KindFreshnessSensitiveRead, effect: EffectRead, idempotency: IdempotencyIdempotent,
		ordering: OrderingPreserve, atomicity: AtomicityPerItem,
		retryAllowed: true, deduplicationAllowed: true, freshnessDependent: true,
	}
}

// IdempotentWrite returns semantics for a write that is safe to repeat.
func IdempotentWrite() Semantics {
	return Semantics{
		kind: KindIdempotentWrite, effect: EffectWrite, idempotency: IdempotencyIdempotent,
		ordering: OrderingPreserve, atomicity: AtomicityPerItem, retryAllowed: true,
	}
}

// NonIdempotentWrite returns semantics for a write that must not be repeated.
func NonIdempotentWrite() Semantics {
	return Semantics{
		kind: KindNonIdempotentWrite, effect: EffectWrite, idempotency: IdempotencyNonIdempotent,
		ordering: OrderingPreserve, atomicity: AtomicityPerItem,
	}
}

// CommutativeAggregation returns semantics for order-independent aggregation.
func CommutativeAggregation() Semantics {
	return Semantics{
		kind: KindCommutativeAggregation, effect: EffectAggregate, idempotency: IdempotencyIdempotent,
		ordering: OrderingCommutative, atomicity: AtomicityPerItem, retryAllowed: true,
	}
}

// OrderedAggregation returns semantics for order-dependent aggregation.
func OrderedAggregation() Semantics {
	return Semantics{
		kind: KindOrderedAggregation, effect: EffectAggregate, idempotency: IdempotencyIdempotent,
		ordering: OrderingPreserve, atomicity: AtomicityPerItem,
	}
}

// AtomicGroup returns semantics for an all-or-nothing group of writes.
func AtomicGroup() Semantics {
	return Semantics{
		kind: KindAtomicGroup, effect: EffectWrite, idempotency: IdempotencyIdempotent,
		ordering: OrderingPreserve, atomicity: AtomicityAllOrNothing,
	}
}

// TransactionBound returns semantics for writes bound to a shared transaction.
func TransactionBound() Semantics {
	return Semantics{
		kind: KindTransactionBound, effect: EffectWrite, idempotency: IdempotencyNonIdempotent,
		ordering: OrderingPreserve, atomicity: AtomicityAllOrNothing,
	}
}

// SessionBound returns semantics for writes bound to a shared session.
func SessionBound() Semantics {
	return Semantics{
		kind: KindSessionBound, effect: EffectWrite, idempotency: IdempotencyNonIdempotent,
		ordering: OrderingPreserve, atomicity: AtomicityPerItem,
	}
}
