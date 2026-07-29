package basic

import (
	"context"
	"errors"
	"testing"

	batchweaver "github.com/Voskan/BatchWeaver"
)

func TestScalarGetUser(t *testing.T) {
	repo := NewRepository()
	u, err := repo.GetUser(context.Background(), 1)
	if err != nil || u.Name != "Ada" {
		t.Fatalf("GetUser(1) = %+v, %v", u, err)
	}
	if _, err := repo.GetUser(context.Background(), 999); !errors.Is(err, ErrUserNotFound) {
		t.Errorf("GetUser(999) error = %v, want ErrUserNotFound", err)
	}
}

func TestBatchGetUsers(t *testing.T) {
	repo := NewRepository()
	req := batchweaver.MustNewBatchRequest([]batchweaver.BatchItem[UserID]{
		batchweaver.NewBatchItem[UserID](1, 1),   // success
		batchweaver.NewBatchItem[UserID](2, 999), // not found
	})
	resp, err := repo.GetUsersBatch(context.Background(), req)
	if err != nil {
		t.Fatalf("GetUsersBatch: %v", err)
	}
	if err := resp.ValidateAgainst(req.IDs()); err != nil {
		t.Fatalf("response does not match request: %v", err)
	}
	out := resp.Outcomes()
	if !out[0].IsSuccess() || out[0].Value.Name != "Ada" {
		t.Errorf("outcome 0 = %+v", out[0])
	}
	if !out[1].IsNotFound() {
		t.Errorf("outcome 1 should be not-found: %+v", out[1])
	}
}

func TestDeclarationValid(t *testing.T) {
	if err := GetUserOperation.Validate(); err != nil {
		t.Errorf("declaration invalid: %v", err)
	}
	if GetUserOperation.Spec().ID() != "users.get" {
		t.Errorf("spec id = %q", GetUserOperation.Spec().ID())
	}
}
