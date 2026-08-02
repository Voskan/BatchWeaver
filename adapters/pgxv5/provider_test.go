package pgxv5_test

import (
	"context"
	"errors"
	"testing"

	"github.com/pashagolub/pgxmock/v4"

	batchweaver "github.com/Voskan/BatchWeaver"
	"github.com/Voskan/BatchWeaver/adapters/pgxv5"
)

type user struct {
	ID   int64
	Name string
}

func TestArrayProviderPreservesOrderDuplicatesAndMissing(t *testing.T) {
	pool, err := pgxmock.NewPool()
	if err != nil {
		t.Fatal(err)
	}
	defer pool.Close()

	query := "SELECT bw_ord, id, name FROM users_by_ids($1)"
	pool.ExpectQuery("SELECT bw_ord").WithArgs([]int64{7, 9, 7}).WillReturnRows(
		pgxmock.NewRows([]string{"bw_ord", "id", "name"}).
			AddRow(1, int64(7), "Ada").
			AddRow(3, int64(7), "Ada"),
	)

	provider := pgxv5.ArrayProvider[int64, user]{
		Queryer: pool,
		Query:   query,
		Decode: func(rows pgxv5.RowScanner) (int, user, error) {
			var ordinal int
			var value user
			err := rows.Scan(&ordinal, &value.ID, &value.Name)
			return ordinal, value, err
		},
	}
	req := batchweaver.MustNewBatchRequest([]batchweaver.BatchItem[int64]{
		batchweaver.NewBatchItem(1, int64(7)),
		batchweaver.NewBatchItem(2, int64(9)),
		batchweaver.NewBatchItem(3, int64(7)),
	})
	resp, err := provider.Execute(context.Background(), req)
	if err != nil {
		t.Fatal(err)
	}
	out := resp.Outcomes()
	if len(out) != 3 || !out[0].Found || out[1].Found || !out[2].Found {
		t.Fatalf("unexpected outcomes: %+v", out)
	}
	if out[0].Value != out[2].Value || out[0].Value.Name != "Ada" {
		t.Fatalf("duplicate mapping changed: %+v", out)
	}
	if err := pool.ExpectationsWereMet(); err != nil {
		t.Fatal(err)
	}
}

func TestArrayProviderRejectsInvalidOrdinal(t *testing.T) {
	pool, err := pgxmock.NewPool()
	if err != nil {
		t.Fatal(err)
	}
	defer pool.Close()
	pool.ExpectQuery("SELECT").WithArgs([]int{1}).WillReturnRows(
		pgxmock.NewRows([]string{"bw_ord", "value"}).AddRow(2, "bad"),
	)
	provider := pgxv5.ArrayProvider[int, string]{
		Queryer: pool,
		Query:   "SELECT bw_ord, value",
		Decode: func(rows pgxv5.RowScanner) (int, string, error) {
			var ordinal int
			var value string
			return ordinal, value, rows.Scan(&ordinal, &value)
		},
	}
	req := batchweaver.MustNewBatchRequest([]batchweaver.BatchItem[int]{batchweaver.NewBatchItem(1, 1)})
	_, err = provider.Execute(context.Background(), req)
	if err == nil || errors.Is(err, pgxv5.ErrMisconfigured) {
		t.Fatalf("invalid ordinal error = %v", err)
	}
}
