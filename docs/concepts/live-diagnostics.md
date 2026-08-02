# Live diagnostics

BatchWeaver publishes diagnostics for the code you are editing, computed from your
unsaved buffers. Batching opportunities appear as informational diagnostics
(code `BW1001`) at the relevant call site; analysis, proof, and contract problems
carry their existing BatchWeaver codes and severities.

Diagnostics are debounced so typing stays responsive, and a stale analysis never
publishes: each edit bumps a generation counter, and only the newest snapshot's
results reach the editor. Every BatchWeaver diagnostic uses the `batchweaver`
source, so it never collides with or clears gopls's compiler diagnostics.
