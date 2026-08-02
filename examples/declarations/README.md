# Declaration examples

These examples show how to declare a BatchWeaver operation using the typed
contracts.

- [basic](basic) — a `Repository` with a scalar `GetUser` method and a native
  `GetUsersBatch` method, connected by a `MustDeclareMethod` declaration.

Declarations are implemented and statically discoverable today. Importing this
package or constructing a declaration has no global side effects. The compiler
can discover eligible call sites, prove a supported strategy, preview the
transformation, and test it through an overlay; this declaration example itself
does not invoke those compiler stages.
