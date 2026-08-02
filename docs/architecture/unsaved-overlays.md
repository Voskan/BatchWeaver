# Unsaved-buffer overlays

While a document is open, its editor buffer — not the file on disk — is
authoritative. BatchWeaver analyzes those unsaved bytes through a `go/packages`
overlay so diagnostics, hover, lenses, and previews reflect exactly what the
developer sees, and nothing is written to disk without an explicit save or apply.

The document store maps each open document's absolute path to its current bytes.
That map is passed as `analysis.Request.Overlay`, which `go/packages` layers over
the on-disk files. The same overlay backs every result of a single request, so a
response is internally consistent. A generation counter and debounce ensure a
result computed from a superseded snapshot is never published.
