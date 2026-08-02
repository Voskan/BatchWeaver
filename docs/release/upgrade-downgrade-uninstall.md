# Upgrade, Downgrade, and Uninstall

## Upgrade

Install the new immutable version, confirm `batchweaver version --json`, keep
the CLI/runtime/extension versions aligned, regenerate proofs and transforms,
and run transformed tests before materialization. Versioned caches are not
assumed compatible across prereleases.

## Downgrade

Disable active adaptive settings and new transformation strategies, revert any
materialized source using its backup manifest, clear only the project's
`.batchweaver-cache` and `.batchweaver-artifacts` directories, install the older
version, regenerate plans, and rerun the full suite. Never reuse a proof from an
incompatible tool/runtime ABI.

## Uninstall

Remove only the explicit binary location returned by `command -v batchweaver`.
Use VS Code's extension UI or `code --uninstall-extension batchweaver.batchweaver`.
Stop the workspace daemon before deleting its project-local state. Remove shell
completion files only from the location where you installed them. Source edits
must be reverted through BatchWeaver or version control before deleting backups.
