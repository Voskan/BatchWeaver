package batchweaver

import (
	"errors"
	"fmt"

	"github.com/Voskan/BatchWeaver/operation"
)

// ErrNilImplementation is returned when a declaration is missing its scalar or
// batch implementation.
var ErrNilImplementation = errors.New("nil implementation")

// FunctionDeclaration connects an operation spec to concrete scalar and batch
// function implementations. It is an immutable-by-convention value with no
// global side effects: constructing one registers nothing.
type FunctionDeclaration[K, V any] struct {
	spec   operation.Spec
	scalar ScalarFunc[K, V]
	batch  BatchFunc[K, V]
}

// DeclareFunction validates the spec and implementations and returns a
// FunctionDeclaration. It returns an error if the spec is invalid or either
// implementation is nil.
func DeclareFunction[K, V any](
	spec operation.Spec,
	scalar ScalarFunc[K, V],
	batch BatchFunc[K, V],
) (FunctionDeclaration[K, V], error) {
	if scalar == nil {
		return FunctionDeclaration[K, V]{}, fmt.Errorf("%w: scalar function is nil", ErrNilImplementation)
	}
	if batch == nil {
		return FunctionDeclaration[K, V]{}, fmt.Errorf("%w: batch function is nil", ErrNilImplementation)
	}
	if err := spec.ValidationError(); err != nil {
		return FunctionDeclaration[K, V]{}, err
	}
	return FunctionDeclaration[K, V]{spec: spec, scalar: scalar, batch: batch}, nil
}

// MustDeclareFunction is like DeclareFunction but panics on error. It is meant
// for package-level declarations, where a failure is a programmer error. The
// panic message includes the operation ID and is deterministic.
func MustDeclareFunction[K, V any](
	spec operation.Spec,
	scalar ScalarFunc[K, V],
	batch BatchFunc[K, V],
) FunctionDeclaration[K, V] {
	d, err := DeclareFunction(spec, scalar, batch)
	if err != nil {
		panic(fmt.Sprintf("batchweaver.MustDeclareFunction(%q): %v", spec.ID(), err))
	}
	return d
}

// Spec returns the operation spec.
func (d FunctionDeclaration[K, V]) Spec() operation.Spec { return d.spec }

// Scalar returns the scalar implementation.
func (d FunctionDeclaration[K, V]) Scalar() ScalarFunc[K, V] { return d.scalar }

// Batch returns the batch implementation.
func (d FunctionDeclaration[K, V]) Batch() BatchFunc[K, V] { return d.batch }

// Validate reports whether the declaration is well-formed.
func (d FunctionDeclaration[K, V]) Validate() error {
	if d.scalar == nil || d.batch == nil {
		return fmt.Errorf("%w: declaration has a nil implementation", ErrNilImplementation)
	}
	return d.spec.ValidationError()
}

// MethodDeclaration connects an operation spec to scalar and batch method
// expressions whose first parameter is the receiver R. It is immutable by
// convention and performs no global registration.
type MethodDeclaration[R, K, V any] struct {
	spec   operation.Spec
	scalar ScalarMethod[R, K, V]
	batch  BatchMethod[R, K, V]
}

// DeclareMethod validates the spec and method expressions and returns a
// MethodDeclaration.
func DeclareMethod[R, K, V any](
	spec operation.Spec,
	scalar ScalarMethod[R, K, V],
	batch BatchMethod[R, K, V],
) (MethodDeclaration[R, K, V], error) {
	if scalar == nil {
		return MethodDeclaration[R, K, V]{}, fmt.Errorf("%w: scalar method is nil", ErrNilImplementation)
	}
	if batch == nil {
		return MethodDeclaration[R, K, V]{}, fmt.Errorf("%w: batch method is nil", ErrNilImplementation)
	}
	if err := spec.ValidationError(); err != nil {
		return MethodDeclaration[R, K, V]{}, err
	}
	return MethodDeclaration[R, K, V]{spec: spec, scalar: scalar, batch: batch}, nil
}

// MustDeclareMethod is like DeclareMethod but panics on error, for package-level
// declarations.
func MustDeclareMethod[R, K, V any](
	spec operation.Spec,
	scalar ScalarMethod[R, K, V],
	batch BatchMethod[R, K, V],
) MethodDeclaration[R, K, V] {
	d, err := DeclareMethod(spec, scalar, batch)
	if err != nil {
		panic(fmt.Sprintf("batchweaver.MustDeclareMethod(%q): %v", spec.ID(), err))
	}
	return d
}

// Spec returns the operation spec.
func (d MethodDeclaration[R, K, V]) Spec() operation.Spec { return d.spec }

// Scalar returns the scalar method expression.
func (d MethodDeclaration[R, K, V]) Scalar() ScalarMethod[R, K, V] { return d.scalar }

// Batch returns the batch method expression.
func (d MethodDeclaration[R, K, V]) Batch() BatchMethod[R, K, V] { return d.batch }

// Validate reports whether the declaration is well-formed.
func (d MethodDeclaration[R, K, V]) Validate() error {
	if d.scalar == nil || d.batch == nil {
		return fmt.Errorf("%w: declaration has a nil implementation", ErrNilImplementation)
	}
	return d.spec.ValidationError()
}
