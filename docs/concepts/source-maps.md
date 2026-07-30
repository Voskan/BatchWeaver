# Source maps

A source map links generated code back to the candidate, proof certificate,
transformation, and generated role it came from. Source maps use workspace-relative
paths and are deterministic.

## Segment

Each segment records the transformed file, a generated line range, a role, and the
originating transformation, candidate, and certificate IDs.

Generated roles include `invariant-binding`, `key-collection`, `batch-call`,
`global-error-check`, `result-reconstruction`, and `scalar-order-replay`.

## Precision

In this stage the mapping is line-granular: segments are located by scanning the
transformed file for each transformation's generated identifiers. Sub-line
precision and full byte ranges are a documented limitation to be refined in a
later stage.

## Schema

The source-map artifact is versioned (`batchweaver.sourcemap/v1alpha1`). See
[the transformation schema reference](../reference/transformation-schema.md).
