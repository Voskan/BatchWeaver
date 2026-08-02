package batchweaver_test

import (
	"context"
	"fmt"

	batchweaver "github.com/Voskan/BatchWeaver"
	"github.com/Voskan/BatchWeaver/operation"
)

type exampleUser struct {
	ID   int
	Name string
}

func exampleLoadUser(_ context.Context, id int) (exampleUser, error) {
	return exampleUser{ID: id, Name: "Ada"}, nil
}

func exampleLoadUsers(
	_ context.Context,
	req batchweaver.BatchRequest[int],
) (batchweaver.BatchResponse[exampleUser], error) {
	values := make([]exampleUser, req.Len())
	for i, item := range req.Items() {
		values[i] = exampleUser{ID: item.Key, Name: "Ada"}
	}
	return batchweaver.OrderedOutcomes(req, values)
}

func ExampleMustDeclareFunction() {
	getUser := batchweaver.MustDeclareFunction(
		operation.MustNewSpec(
			operation.MustParseID("users.get"),
			operation.ReadOnly(),
			operation.WithOrderedResults(),
			operation.WithRequestScope(),
		),
		exampleLoadUser,
		exampleLoadUsers,
	)

	fmt.Println(getUser.Spec().ID())
	// Output: users.get
}
