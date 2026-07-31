# Compiler driver threat model

The `-toolexec` driver and runtime lowering treat compiler arguments, the
environment, and generated code as sensitive, and constrain their behavior.

## Compiler driver

- Tools run with `exec.CommandContext` and an explicit argument array — never a
  shell, so there is no argument interpolation.
- The full environment is never logged; a private marker prevents recursive
  `-toolexec` invocation and BatchWeaver-internal variables are stripped from
  child processes.
- The child exit code is preserved and reported distinctly (exit code 6), so a
  tool failure is never masked as an internal error.

## Generated code

- Generated bridges contain no secret constants and no reflection on the call
  path. They carry the standard generated-code header and live in the source
  package as normal Go files.

## Source maps and artifacts

- Source maps and plans use workspace-relative paths and never contain raw keys
  or runtime values.
- Plans, overlays, and backups remain under the ignored `.batchweaver/`
  directory and are never uploaded or transmitted.

## Runtime verification

- Shadow verification is restricted to read-only operations and is off by
  default, so it never causes unauthorized duplicate external access.

## Paths

- The driver and materialization use the Prompt 06 path policy: writes are
  confined to the workspace state directory and, on explicit materialization, to
  files inside writable workspace modules.
