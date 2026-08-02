# Tutorial: From Scalar N+1 to Verified Batch Execution

Use `examples/static-prefetch`, which has a deterministic in-memory backend and
no credentials.

1. Read the scalar loop and explicit scalar/batch operation declaration.
2. Build the CLI with `make build`.
3. Scan the example: `bin/batchweaver scan ./examples/static-prefetch/...`.
4. Run proof and inspect its evidence: `bin/batchweaver prove
   ./examples/static-prefetch/...`.
5. Create a plan and preview its diff with `transform plan` and `transform diff`.
6. Run `bin/batchweaver test -- -race ./examples/static-prefetch/...`; this uses
   an overlay and does not modify source.
7. Compare scalar/batch values, errors, ordering, cancellation, and backend call
   counts in the example test.
8. If evaluating materialization, use a disposable branch, inspect the backup,
   run the full application suite, then demonstrate `transform revert`.

The tutorial does not prove that an unrelated application is safe or faster.
Its purpose is to make the evidence and rollback workflow reproducible.
