# Declaration examples

These examples show how to declare a BatchWeaver operation using the typed
contracts.

- [basic](basic) — a `Repository` with a scalar `GetUser` method and a native
  `GetUsersBatch` method, connected by a `MustDeclareMethod` declaration.

**Declaration is implemented today. Automatic interception of scalar calls is
not.** These examples demonstrate the data and type contracts only; later
compiler prompts will discover eligible call sites and transform them.
