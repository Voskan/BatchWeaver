package operation

import "errors"

// Kind classifies an operation's batching-relevant semantics. It drives many
// downstream validation rules, so the zero value is Unknown and is never valid
// in a complete spec.
type Kind uint8

const (
	// KindUnknown is the zero value and is invalid in a complete spec.
	KindUnknown Kind = iota
	// KindReadOnly is a read that does not observe or affect mutable state
	// beyond what it returns and tolerates reordering when declared.
	KindReadOnly
	// KindFreshnessSensitiveRead is a read whose result depends on a freshness
	// or consistency level and therefore constrains coalescing.
	KindFreshnessSensitiveRead
	// KindIdempotentWrite is a write that may be safely repeated without
	// changing the observable outcome.
	KindIdempotentWrite
	// KindNonIdempotentWrite is a write that must not be repeated automatically.
	KindNonIdempotentWrite
	// KindCommutativeAggregation combines inputs where order does not affect the
	// result.
	KindCommutativeAggregation
	// KindOrderedAggregation combines inputs where order affects the result.
	KindOrderedAggregation
	// KindAtomicGroup requires all items to succeed or fail together.
	KindAtomicGroup
	// KindTransactionBound requires all items to share a transaction context.
	KindTransactionBound
	// KindSessionBound requires all items to share a session context.
	KindSessionBound
)

// ErrInvalidKind is returned when parsing or validating an unknown kind.
var ErrInvalidKind = errors.New("invalid operation kind")

var kindNames = []string{
	"unknown",
	"read-only",
	"freshness-sensitive-read",
	"idempotent-write",
	"non-idempotent-write",
	"commutative-aggregation",
	"ordered-aggregation",
	"atomic-group",
	"transaction-bound",
	"session-bound",
}

// String returns the canonical hyphenated name of the kind.
func (k Kind) String() string { return enumString(k, kindNames) }

// Valid reports whether k is a defined kind other than Unknown.
func (k Kind) Valid() bool { return k != KindUnknown && int(k) < len(kindNames) }

// Effect returns the externally observable effect implied by the kind.
func (k Kind) Effect() Effect {
	switch k {
	case KindReadOnly, KindFreshnessSensitiveRead:
		return EffectRead
	case KindIdempotentWrite, KindNonIdempotentWrite, KindAtomicGroup,
		KindTransactionBound, KindSessionBound:
		return EffectWrite
	case KindCommutativeAggregation, KindOrderedAggregation:
		return EffectAggregate
	default:
		return EffectUnknown
	}
}

// ParseKind resolves the canonical name of a kind.
func ParseKind(s string) (Kind, error) { return parseEnum[Kind](s, kindNames, ErrInvalidKind) }

// MarshalText implements encoding.TextMarshaler.
func (k Kind) MarshalText() ([]byte, error) { return marshalEnum(k, kindNames, ErrInvalidKind) }

// UnmarshalText implements encoding.TextUnmarshaler.
func (k *Kind) UnmarshalText(data []byte) error {
	v, err := ParseKind(string(data))
	if err != nil {
		return err
	}
	*k = v
	return nil
}

// Effect is the externally observable effect of an operation.
type Effect uint8

const (
	// EffectUnknown is the zero value and is invalid in a complete spec.
	EffectUnknown Effect = iota
	// EffectRead observes state without modifying it.
	EffectRead
	// EffectWrite modifies state.
	EffectWrite
	// EffectAggregate combines multiple inputs into a result.
	EffectAggregate
)

// ErrInvalidEffect is returned when parsing or validating an unknown effect.
var ErrInvalidEffect = errors.New("invalid effect")

var effectNames = []string{"unknown", "read", "write", "aggregate"}

// String returns the canonical name of the effect.
func (e Effect) String() string { return enumString(e, effectNames) }

// Valid reports whether e is a defined effect other than Unknown.
func (e Effect) Valid() bool { return e != EffectUnknown && int(e) < len(effectNames) }

// ParseEffect resolves the canonical name of an effect.
func ParseEffect(s string) (Effect, error) {
	return parseEnum[Effect](s, effectNames, ErrInvalidEffect)
}

// MarshalText implements encoding.TextMarshaler.
func (e Effect) MarshalText() ([]byte, error) { return marshalEnum(e, effectNames, ErrInvalidEffect) }

// UnmarshalText implements encoding.TextUnmarshaler.
func (e *Effect) UnmarshalText(data []byte) error {
	v, err := ParseEffect(string(data))
	if err != nil {
		return err
	}
	*e = v
	return nil
}
