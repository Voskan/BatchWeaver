package operation

import (
	"errors"
	"fmt"
	"strings"
)

// Scope names the boundary within which logical requests may be coalesced.
type Scope uint8

const (
	// ScopeRequest confines batching to a single inbound request. It is the
	// safe default.
	ScopeRequest Scope = iota
	// ScopeJob confines batching to a background job.
	ScopeJob
	// ScopeGraphQLOperation confines batching to one GraphQL operation.
	ScopeGraphQLOperation
	// ScopeSession confines batching to a session.
	ScopeSession
	// ScopeTransaction confines batching to a transaction.
	ScopeTransaction
	// ScopeProcess permits batching across the whole process; it is the most
	// dangerous scope and requires strong isolation.
	ScopeProcess
	// ScopeCustom defers scope definition to a later mechanism.
	ScopeCustom
)

// ErrInvalidScope is returned for an unknown scope.
var ErrInvalidScope = errors.New("invalid scope")

var scopeNames = []string{"request", "job", "graphql-operation", "session", "transaction", "process", "custom"}

// String returns the canonical name of the scope.
func (s Scope) String() string { return enumString(s, scopeNames) }

// Valid reports whether s is a defined scope.
func (s Scope) Valid() bool { return int(s) < len(scopeNames) }

// ParseScope resolves the canonical name of a scope.
func ParseScope(v string) (Scope, error) { return parseEnum[Scope](v, scopeNames, ErrInvalidScope) }

// MarshalText implements encoding.TextMarshaler.
func (s Scope) MarshalText() ([]byte, error) { return marshalEnum(s, scopeNames, ErrInvalidScope) }

// UnmarshalText implements encoding.TextUnmarshaler.
func (s *Scope) UnmarshalText(data []byte) error {
	v, err := ParseScope(string(data))
	if err != nil {
		return err
	}
	*s = v
	return nil
}

// PartitionDimension is a validated partitioning key. Well-known dimensions
// have exported constants; custom dimensions are created with CustomDimension
// and must use a validated name that does not collide with a reserved one.
type PartitionDimension string

// Well-known partition dimensions.
const (
	DimensionReceiver          PartitionDimension = "receiver"
	DimensionTenant            PartitionDimension = "tenant"
	DimensionAuthorization     PartitionDimension = "authorization"
	DimensionTransaction       PartitionDimension = "transaction"
	DimensionSession           PartitionDimension = "session"
	DimensionConsistency       PartitionDimension = "consistency"
	DimensionRegion            PartitionDimension = "region"
	DimensionEncryptionContext PartitionDimension = "encryption-context"
	DimensionDeadlineClass     PartitionDimension = "deadline-class"
)

// ErrInvalidDimension is returned when a PartitionDimension fails validation.
var ErrInvalidDimension = errors.New("invalid partition dimension")

// reservedDimensionNames are names custom dimensions must not use.
var reservedDimensionNames = map[string]struct{}{
	"receiver": {}, "tenant": {}, "authorization": {}, "transaction": {},
	"session": {}, "consistency": {}, "region": {}, "encryption-context": {},
	"deadline-class": {}, "custom": {}, "unknown": {},
}

// knownDimensions is the set of well-known dimension values.
var knownDimensions = map[PartitionDimension]struct{}{
	DimensionReceiver: {}, DimensionTenant: {}, DimensionAuthorization: {},
	DimensionTransaction: {}, DimensionSession: {}, DimensionConsistency: {},
	DimensionRegion: {}, DimensionEncryptionContext: {}, DimensionDeadlineClass: {},
}

// CustomDimension validates name and returns it as a custom PartitionDimension.
// The name must be lowercase, start with a letter, use only letters, digits,
// and single hyphens, and must not collide with a reserved dimension name.
func CustomDimension(name string) (PartitionDimension, error) {
	if err := validateDimensionName(name); err != nil {
		return "", err
	}
	if _, reserved := reservedDimensionNames[name]; reserved {
		return "", fmt.Errorf("%w: %q is reserved", ErrInvalidDimension, name)
	}
	return PartitionDimension(name), nil
}

// validateDimensionName checks the lexical form of a dimension name.
func validateDimensionName(name string) error {
	if name == "" {
		return fmt.Errorf("%w: dimension name is empty", ErrInvalidDimension)
	}
	if len(name) > maxSegmentBytes {
		return fmt.Errorf("%w: dimension name exceeds %d bytes", ErrInvalidDimension, maxSegmentBytes)
	}
	for i := 0; i < len(name); i++ {
		c := name[i]
		switch {
		case c >= 'a' && c <= 'z':
		case i > 0 && (c >= '0' && c <= '9' || c == '-'):
		default:
			return fmt.Errorf("%w: %q has invalid character %q", ErrInvalidDimension, name, c)
		}
	}
	if strings.HasSuffix(name, "-") || strings.Contains(name, "--") {
		return fmt.Errorf("%w: %q has a misplaced hyphen", ErrInvalidDimension, name)
	}
	return nil
}

// Validate reports whether the dimension is well-known or a valid custom name.
func (d PartitionDimension) Validate() error {
	if _, ok := knownDimensions[d]; ok {
		return nil
	}
	if _, reserved := reservedDimensionNames[string(d)]; reserved {
		return fmt.Errorf("%w: %q is reserved and not a well-known dimension", ErrInvalidDimension, string(d))
	}
	return validateDimensionName(string(d))
}

// IsWellKnown reports whether the dimension is one of the predefined dimensions.
func (d PartitionDimension) IsWellKnown() bool {
	_, ok := knownDimensions[d]
	return ok
}

// String returns the dimension name.
func (d PartitionDimension) String() string { return string(d) }

// ReceiverPolicy describes whether operations on different receiver values may
// share a batch.
type ReceiverPolicy uint8

const (
	// ReceiverSame requires all items in a batch to share the same receiver.
	ReceiverSame ReceiverPolicy = iota
	// ReceiverAny permits items with different receivers to share a batch.
	ReceiverAny
)

// ErrInvalidReceiverPolicy is returned for an unknown receiver policy.
var ErrInvalidReceiverPolicy = errors.New("invalid receiver policy")

var receiverPolicyNames = []string{"same-receiver", "any-receiver"}

// String returns the canonical name of the receiver policy.
func (r ReceiverPolicy) String() string { return enumString(r, receiverPolicyNames) }

// Valid reports whether r is a defined receiver policy.
func (r ReceiverPolicy) Valid() bool { return int(r) < len(receiverPolicyNames) }

// MissingDimensionBehavior describes what happens when a required partition
// dimension cannot be extracted at runtime (extraction is implemented later).
type MissingDimensionBehavior uint8

const (
	// MissingDimensionError fails the request when a required dimension is
	// absent. This is the safe default.
	MissingDimensionError MissingDimensionBehavior = iota
	// MissingDimensionReject rejects batching but does not fail the request.
	MissingDimensionReject
)

var missingDimensionNames = []string{"error", "reject"}

// String returns the canonical name of the missing-dimension behavior.
func (m MissingDimensionBehavior) String() string { return enumString(m, missingDimensionNames) }

// Valid reports whether m is a defined missing-dimension behavior.
func (m MissingDimensionBehavior) Valid() bool { return int(m) < len(missingDimensionNames) }

// ErrInvalidPartitionContract is returned when a PartitionContract fails
// validation.
var ErrInvalidPartitionContract = errors.New("invalid partition contract")

// PartitionContract describes how callers are isolated into separate batches.
// Its security invariants prevent semantically or security-incompatible callers
// from sharing a batch. Extraction of dimension values and runtime enforcement
// are implemented in later prompts.
type PartitionContract struct {
	scope               Scope
	required            []PartitionDimension
	optional            []PartitionDimension
	crossRootScope      bool
	receiverPolicy      ReceiverPolicy
	missingDimension    MissingDimensionBehavior
	maxActivePartitions int
	publicContextIndep  bool
}

// NewPartitionContract returns a partition contract for scope with the given
// required and optional dimensions. Input slices are copied.
func NewPartitionContract(scope Scope, required, optional []PartitionDimension) PartitionContract {
	return PartitionContract{
		scope:            scope,
		required:         append([]PartitionDimension(nil), required...),
		optional:         append([]PartitionDimension(nil), optional...),
		receiverPolicy:   ReceiverSame,
		missingDimension: MissingDimensionError,
	}
}

// DefaultPartitionContract returns the conservative default: request scope, no
// required dimensions, same-receiver batching.
func DefaultPartitionContract() PartitionContract {
	return NewPartitionContract(ScopeRequest, nil, nil)
}

// Scope returns the partition scope.
func (c PartitionContract) Scope() Scope { return c.scope }

// Required returns a copy of the required dimensions.
func (c PartitionContract) Required() []PartitionDimension {
	return append([]PartitionDimension(nil), c.required...)
}

// Optional returns a copy of the optional dimensions.
func (c PartitionContract) Optional() []PartitionDimension {
	return append([]PartitionDimension(nil), c.optional...)
}

// CrossRootScopeAllowed reports whether cross-root-scope batching is allowed.
func (c PartitionContract) CrossRootScopeAllowed() bool { return c.crossRootScope }

// ReceiverPolicy returns the receiver-identity policy.
func (c PartitionContract) ReceiverPolicy() ReceiverPolicy { return c.receiverPolicy }

// MissingDimension returns the missing-dimension behavior.
func (c PartitionContract) MissingDimension() MissingDimensionBehavior { return c.missingDimension }

// MaxActivePartitions returns the active-partition hint (0 means unbounded hint).
func (c PartitionContract) MaxActivePartitions() int { return c.maxActivePartitions }

// PublicContextIndependent reports whether the operation is explicitly public
// and independent of caller context, relaxing process-scope isolation.
func (c PartitionContract) PublicContextIndependent() bool { return c.publicContextIndep }

// WithCrossRootScope returns a copy that permits cross-root-scope batching.
func (c PartitionContract) WithCrossRootScope() PartitionContract {
	c.crossRootScope = true
	return c
}

// WithReceiverPolicy returns a copy using the given receiver policy.
func (c PartitionContract) WithReceiverPolicy(p ReceiverPolicy) PartitionContract {
	c.receiverPolicy = p
	return c
}

// WithMaxActivePartitions returns a copy with the given active-partition hint.
func (c PartitionContract) WithMaxActivePartitions(n int) PartitionContract {
	c.maxActivePartitions = n
	return c
}

// WithPublicContextIndependent returns a copy marked public and
// context-independent, which relaxes the process-scope isolation requirement.
func (c PartitionContract) WithPublicContextIndependent() PartitionContract {
	c.publicContextIndep = true
	return c
}

// requires reports whether dim is in the required set.
func (c PartitionContract) requires(dim PartitionDimension) bool {
	for _, d := range c.required {
		if d == dim {
			return true
		}
	}
	return false
}

// Validate reports whether the partition contract is internally consistent and
// satisfies scope-independent security invariants.
func (c PartitionContract) Validate() error {
	if !c.scope.Valid() {
		return fmt.Errorf("%w: %w", ErrInvalidPartitionContract, ErrInvalidScope)
	}
	if !c.receiverPolicy.Valid() {
		return fmt.Errorf("%w: %w", ErrInvalidPartitionContract, ErrInvalidReceiverPolicy)
	}
	if !c.missingDimension.Valid() {
		return fmt.Errorf("%w: invalid missing-dimension behavior", ErrInvalidPartitionContract)
	}
	if c.maxActivePartitions < 0 {
		return fmt.Errorf("%w: max active partitions is negative", ErrInvalidPartitionContract)
	}
	seen := make(map[PartitionDimension]struct{})
	for _, d := range append(append([]PartitionDimension(nil), c.required...), c.optional...) {
		if err := d.Validate(); err != nil {
			return fmt.Errorf("%w: %w", ErrInvalidPartitionContract, err)
		}
		if _, dup := seen[d]; dup {
			return fmt.Errorf("%w: dimension %q appears more than once", ErrInvalidPartitionContract, d)
		}
		seen[d] = struct{}{}
	}
	if c.scope == ScopeProcess && !c.publicContextIndep {
		if !c.requires(DimensionAuthorization) || !c.requires(DimensionTenant) {
			return fmt.Errorf("%w: process scope requires tenant and authorization dimensions unless public and context-independent", ErrInvalidPartitionContract)
		}
	}
	return nil
}
