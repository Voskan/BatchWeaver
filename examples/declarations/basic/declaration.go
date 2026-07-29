package basic

import (
	batchweaver "github.com/Voskan/BatchWeaver"
	"github.com/Voskan/BatchWeaver/operation"
)

// GetUserOperation is the canonical, statically discoverable declaration shape:
// a package-level value produced by MustDeclareMethod from an operation spec and
// two method expressions. A future analyzer can find declarations like this
// without executing any code. Type parameters are inferred: the receiver is
// *Repository, the key is UserID, and the value is User.
var GetUserOperation = batchweaver.MustDeclareMethod(
	operation.MustNewSpec(
		operation.MustParseID("users.get"),
		operation.ReadOnly(),
		operation.WithOrderedResults(),
		operation.WithRequestScope(),
	),
	(*Repository).GetUser,
	(*Repository).GetUsersBatch,
)
