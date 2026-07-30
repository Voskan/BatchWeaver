# Transformation threat model

The transformation stage treats analyzed source, configuration, and proof metadata
as untrusted input and constrains its own filesystem and process behavior.

## Filesystem

- The workspace root is canonicalized; writes are confined to the ignored
  `.batchweaver/` state directory and, only during explicit materialization, to
  files inside writable workspace modules.
- Materialization uses atomic writes (temp file + rename) and takes a full backup
  first; it never leaves a half-written source file.
- Module-cache and third-party files are never edited; they can only be
  transformed through overlays.

## Command execution

- The Go tool is run with `exec.CommandContext` and an explicit argument array —
  never through a shell.
- BatchWeaver-specific environment variables are stripped from the child to prevent
  recursive wrapper invocation.
- The child exit code is preserved and reported distinctly (exit code 6) so a Go
  command failure is never masked as an internal error.

## Untrusted metadata

- Proof and configuration metadata are declarative and are never executed.
- Certificates are validated (schema, decision, strategy eligibility, digests,
  assumptions) before any rewrite; a certificate is never accepted by ID alone.

## Artifact privacy

- Plans, overlays, and backups are local under `.batchweaver/` and are never
  uploaded or transmitted.
- Deterministic artifacts use workspace-relative paths and exclude timestamps and
  host-specific absolute paths.

## Limits

Planning is bounded (files, transformations, overlay contents) and honors context
cancellation; exceeding a limit yields a failed plan with a reason, never a partial
unsafe plan.
