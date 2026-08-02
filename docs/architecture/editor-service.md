# Editor service

The editor service (`internal/editor`) is BatchWeaver's editor-agnostic layer. It
turns a snapshot-bound analysis of unsaved editor buffers into deterministic,
LSP-shaped results — diagnostics, hover, code lenses, and code actions — without
depending on any editor, the LSP transport, or the proxy.

It never applies edits, runs project code, or writes source. Every result is a
projection of a single analysis snapshot over an overlay of the currently open
buffers, so the same buffers always yield the same results.

## Inputs and outputs

`Service.Analyze(ctx, overlay)` first asks a compatible workspace daemon for the
content-addressed snapshot and safely falls back to `analysis.Analyze` when the
daemon is absent or unavailable. The overlay (unsaved buffers) is authoritative
in both paths. The returned `Result` indexes call sites, operations, and
candidates. `Diagnostics`, `Hover`, `CodeLens`, and `CodeActions` map that result
to protocol types using the document's canonical UTF-16 mapper. `ScanSummary`,
`OperationGraphText`, `PreviewText`, and `DoctorText` back the workspace commands.

## Coordinates

Analysis reports `path:line:col` locations with 1-based byte columns. The editor
service converts these to LSP UTF-16 ranges through the single `documents.Mapper`
so every feature agrees on positions, including multibyte and emoji text.
