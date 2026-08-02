# Manual Batching Migration

Keep a proven hand-written batch provider. Describe its scalar contract, batch
mapping, partition identity, scheduling bounds, cancellation, retry, and error
semantics in a BatchWeaver declaration. Bind it through the typed runtime and
start with scan/proof/overlay tests.

Do not replace backend-specific chunking or transaction rules unless equivalent
evidence exists. Compare the original and BatchWeaver paths with deterministic
keys, duplicates, missing results, faults, cancellation, and load. Roll back by
selecting the original provider or exact scalar fallback.
