package adapter

import (
	"context"
	"crypto/sha256"
	"encoding/hex"
	"errors"
	"fmt"
	"io"

	batchweaver "github.com/Voskan/BatchWeaver"
)

// VerificationSchema versions the verification contract artifact.
const VerificationSchema = "batchweaver.verification/v1alpha1"

// CaseResult is the outcome of one verification case.
type CaseResult struct {
	Name   string `json:"name"`
	Passed bool   `json:"passed"`
	Detail string `json:"detail,omitempty"`
}

// VerificationContract is the deterministic artifact of comparing scalar and
// batch execution for an operation. It contains no credentials or raw values.
type VerificationContract struct {
	SchemaVersion string       `json:"schema_version"`
	Operation     string       `json:"operation"`
	Adapter       string       `json:"adapter"`
	Cases         []CaseResult `json:"cases"`
	Passed        bool         `json:"passed"`
	Digest        string       `json:"digest"`
}

// VerifyCase is one comparison scenario, identified by a name and its input keys.
type VerifyCase[K any] struct {
	Name string
	Keys []K
}

// ScalarFn executes one scalar read.
type ScalarFn[K, V any] func(context.Context, K) (V, error)

// BatchFn executes an ordered batch read, returning one outcome per input key.
type BatchFn[K, V any] func(context.Context, []K) ([]batchweaver.Outcome[V], error)

// VerifyReadOnly compares scalar and batch execution across cases and returns a
// deterministic verification contract. Values are compared with equal; errors are
// compared by nilness and by errors.Is against the scalar error. It never shadows
// writes; callers must only pass read-only operations.
func VerifyReadOnly[K any, V any](
	ctx context.Context,
	operation, adapterID string,
	scalar ScalarFn[K, V],
	batch BatchFn[K, V],
	equal func(V, V) bool,
	cases []VerifyCase[K],
) VerificationContract {
	vc := VerificationContract{SchemaVersion: VerificationSchema, Operation: operation, Adapter: adapterID, Passed: true}
	for _, c := range cases {
		res := verifyCase(ctx, scalar, batch, equal, c)
		if !res.Passed {
			vc.Passed = false
		}
		vc.Cases = append(vc.Cases, res)
	}
	vc.Digest = verificationDigest(vc)
	return vc
}

func verifyCase[K, V any](ctx context.Context, scalar ScalarFn[K, V], batch BatchFn[K, V], equal func(V, V) bool, c VerifyCase[K]) CaseResult {
	// Scalar reference outcomes, in order.
	scalarVals := make([]V, len(c.Keys))
	scalarErrs := make([]error, len(c.Keys))
	for i, k := range c.Keys {
		scalarVals[i], scalarErrs[i] = scalar(ctx, k)
	}
	outcomes, err := batch(ctx, c.Keys)
	if err != nil {
		// A global batch error is acceptable only if every scalar call also failed.
		for _, se := range scalarErrs {
			if se == nil {
				return CaseResult{Name: c.Name, Passed: false, Detail: "batch returned a global error but a scalar call succeeded"}
			}
		}
		return CaseResult{Name: c.Name, Passed: true}
	}
	if len(outcomes) != len(c.Keys) {
		return CaseResult{Name: c.Name, Passed: false, Detail: fmt.Sprintf("batch returned %d outcomes for %d keys", len(outcomes), len(c.Keys))}
	}
	for i := range c.Keys {
		if (scalarErrs[i] == nil) != (outcomes[i].Err == nil) {
			return CaseResult{Name: c.Name, Passed: false, Detail: fmt.Sprintf("item %d error presence differs", i)}
		}
		if scalarErrs[i] != nil {
			if !errors.Is(outcomes[i].Err, scalarErrs[i]) && !errors.Is(scalarErrs[i], outcomes[i].Err) {
				return CaseResult{Name: c.Name, Passed: false, Detail: fmt.Sprintf("item %d error identity differs", i)}
			}
			continue
		}
		if !outcomes[i].Found {
			return CaseResult{Name: c.Name, Passed: false, Detail: fmt.Sprintf("item %d: scalar found a value but batch did not", i)}
		}
		if !equal(scalarVals[i], outcomes[i].Value) {
			return CaseResult{Name: c.Name, Passed: false, Detail: fmt.Sprintf("item %d value differs", i)}
		}
	}
	return CaseResult{Name: c.Name, Passed: true}
}

// verificationDigest returns a deterministic digest of a contract's case results.
func verificationDigest(vc VerificationContract) string {
	h := sha256.New()
	w := func(s string) { _, _ = io.WriteString(h, s); _, _ = h.Write([]byte{0}) }
	w(vc.SchemaVersion)
	w(vc.Operation)
	w(vc.Adapter)
	for _, c := range vc.Cases {
		w(c.Name)
		if c.Passed {
			w("pass")
		} else {
			w("fail")
		}
	}
	return "bwcontract_" + hex.EncodeToString(h.Sum(nil))[:16]
}
