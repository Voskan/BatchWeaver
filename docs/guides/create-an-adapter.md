# Create an adapter

An adapter is described by a versioned manifest and provides a runtime batch
provider that satisfies the typed runtime contract.

1. Declare a `Manifest` with a stable adapter ID, version, runtime ABI, and only
   implemented capabilities from the closed vocabulary. Compute its digest.
2. Implement a runtime provider with the signature
   `Execute(ctx, batchweaver.BatchRequest[K]) (batchweaver.BatchResponse[V], error)`,
   preserving order, duplicates, missing outcomes, and error identity.
3. For SQL adapters, reuse the exact/composite-key parser and synthesizer. A
   join must remain within its bounded grammar and carry explicit at-most-one
   evidence; otherwise accept an explicit batch provider.
4. Verify the adapter with the contract-verification harness and record the
   contract digest.

Compile-time adapter code must never open a backend connection; runtime code must
never import `go/ast` or `go/types`.
